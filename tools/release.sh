#!/usr/bin/env bash
set -euo pipefail

# Release script for scapectl.
# Usage: ./tools/release.sh [version]
#
# If version is omitted, auto-increments the patch number from the latest tag.
#
# Builds first, then tags, pushes, and publishes the release. The exception is
# a run triggered by a release that was published from the GitHub Releases
# page: the tag and release already exist there, so this only builds the
# assets and attaches them.
#
# Steps are idempotent: re-running after a partial failure skips work that's done.

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

MODULE="./cmd/scapectl"
LDFLAGS_BASE="-s -w"

# ── Preflight ──

command -v go >/dev/null 2>&1 || { echo "Error: go is not installed." >&2; exit 1; }
command -v gh >/dev/null 2>&1 || { echo "Error: gh is not installed. Install with: brew install gh" >&2; exit 1; }

# ── Resolve version ──

if [[ -n "${1:-}" ]]; then
    VERSION="${1#v}"
else
    LATEST=$(git tag --list 'v*' --sort=-version:refname | head -1)
    if [[ -z "$LATEST" ]]; then
        VERSION="0.0.1"
    else
        VER="${LATEST#v}"
        MAJOR="${VER%%.*}"
        REST="${VER#*.}"
        MINOR="${REST%%.*}"
        PATCH="${REST#*.}"
        PATCH=$((PATCH + 1))
        VERSION="${MAJOR}.${MINOR}.${PATCH}"
    fi
fi

TAG="v${VERSION}"
echo "==> Version: ${VERSION} (tag: ${TAG})"

# Hand the resolved version back to the workflow. The cask bump step needs it,
# and on a "Run workflow" run there's no tag in GITHUB_REF to read it from.
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
    echo "version=${VERSION}" >> "${GITHUB_OUTPUT}"
fi

# ── Who owns the tag and release? ──
#
# A run triggered by release:published is downstream of a release GitHub
# already created, so it must not tag or create anything. Every other run
# (local, or "Run workflow" from the Actions tab) owns both.

if [[ "${GITHUB_EVENT_NAME:-}" == "release" ]]; then
    OWNS_RELEASE=0
else
    OWNS_RELEASE=1
fi

# ── Preflight ──
#
# Do these before building so a failed build never leaves an orphan tag
# or a half-published release behind. A re-run after partial failure is
# allowed as long as no release was published for this version yet.

if [[ -z "${GITHUB_ACTIONS:-}" && -n "$(git status --porcelain)" ]]; then
    echo "Error: working tree is not clean. Commit or stash changes first." >&2
    exit 1
fi

if [[ "${OWNS_RELEASE}" == "1" ]] && gh release view "${TAG}" >/dev/null 2>&1; then
    echo "Error: release ${TAG} already exists. Pick a different version or delete the existing release." >&2
    exit 1
fi

# ── Build setup ──

BUILD_DIR=$(mktemp -d)
trap 'rm -rf "$BUILD_DIR"' EXIT

echo "==> Building macOS (arm64)..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
    go build -ldflags "${LDFLAGS_BASE} -X main.version=${VERSION}" \
    -o "${BUILD_DIR}/scapectl-macos-binary" ${MODULE}

echo "==> Bundling ScapeCtl.app..."
./tools/bundle-mac.sh "${BUILD_DIR}/scapectl-macos-binary" "${VERSION}" "${BUILD_DIR}"

(cd "${BUILD_DIR}" && zip -qr Mac_ScapeCtl.zip ScapeCtl.app)

echo "==> Building Linux (amd64)..."
LINUX_DIR="${BUILD_DIR}/ScapeCtl"
mkdir -p "${LINUX_DIR}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "${LDFLAGS_BASE} -X main.version=${VERSION}" \
    -o "${LINUX_DIR}/scapectl" ${MODULE}
cp config.example.toml 50-fractal.rules "${LINUX_DIR}/"
tar -czf "${BUILD_DIR}/Linux_ScapeCtl.tar.gz" -C "${BUILD_DIR}" ScapeCtl

echo "==> Building Windows (amd64)..."
WIN_DIR="${BUILD_DIR}/win"
mkdir -p "${WIN_DIR}"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
    go build -ldflags "${LDFLAGS_BASE} -H windowsgui -X main.version=${VERSION}" \
    -o "${WIN_DIR}/scapectl.exe" ${MODULE}
cp config.example.toml scripts/notify.ps1 "${WIN_DIR}/"
(cd "${WIN_DIR}" && zip -qr "${BUILD_DIR}/Win_ScapeCtl.zip" .)

# ── Tag ──
#
# Push the tag only after a successful build. Each step is skipped if it
# already happened, so the script can be re-run after a partial failure.

if [[ "${OWNS_RELEASE}" == "1" ]]; then
    if git rev-parse -q --verify "refs/tags/${TAG}" >/dev/null; then
        echo "==> Tag ${TAG} already exists locally, skipping git tag."
    else
        echo "==> Creating tag ${TAG}..."
        git tag "${TAG}"
    fi
    if git ls-remote --exit-code --tags origin "${TAG}" >/dev/null 2>&1; then
        echo "==> Tag ${TAG} already on origin, skipping push."
    else
        git push origin "${TAG}"
    fi
fi

# ── Create GitHub Release ──
#
# When the release already exists -- published from the Releases page, or left
# behind by a partly failed run -- attach the freshly built assets to it
# instead. --clobber replaces any assets already there, so re-runs are safe.

ASSETS=(
    "${BUILD_DIR}/Mac_ScapeCtl.zip"
    "${BUILD_DIR}/Linux_ScapeCtl.tar.gz"
    "${BUILD_DIR}/Win_ScapeCtl.zip"
)

if gh release view "${TAG}" >/dev/null 2>&1; then
    echo "==> Release ${TAG} already exists, uploading assets..."
    gh release upload "${TAG}" --clobber "${ASSETS[@]}"
else
    echo "==> Creating GitHub release ${TAG}..."
    gh release create "${TAG}" \
        --title "${TAG}" \
        --generate-notes \
        "${ASSETS[@]}"
fi

echo "==> Done! Release ${TAG} ready."

# ── Bump Homebrew cask ──
# Only runs locally. In CI the "Bump Homebrew cask" step in
# .github/workflows/release.yml does this against main instead.

if [[ -z "${GITHUB_ACTIONS:-}" && -f Casks/scapectl.rb ]]; then
    echo "==> Bumping Homebrew cask to ${VERSION}..."
    ZIP_SHA=$(shasum -a 256 "${BUILD_DIR}/Mac_ScapeCtl.zip" | awk '{print $1}')
    sed -i.bak -E \
        -e "s|version \"[0-9]+\.[0-9]+\.[0-9]+\"|version \"${VERSION}\"|" \
        -e "s|sha256 \"[0-9a-f]{64}\"|sha256 \"${ZIP_SHA}\"|" \
        Casks/scapectl.rb
    rm -f Casks/scapectl.rb.bak
    if ! git diff --quiet Casks/scapectl.rb; then
        git add Casks/scapectl.rb
        git commit -m "Bump cask to ${TAG}"
        git push origin HEAD
        echo "==> Cask bumped and pushed."
    fi
fi
