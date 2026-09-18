#!/usr/bin/env bash
set -euo pipefail

# Usage: ./tools/bundle-mac.sh <binary> <version> <out-dir>

BINARY="$1"
VERSION="$2"
OUT_DIR="$3"

APP_DIR="${OUT_DIR}/ScapeCtl.app/Contents"
rm -rf "${OUT_DIR}/ScapeCtl.app"
mkdir -p "${APP_DIR}/MacOS" "${APP_DIR}/Resources"
cp "${BINARY}" "${APP_DIR}/MacOS/scapectl"
cp config.example.toml "${APP_DIR}/Resources/"

# Compile the light/dark app icon.
xcrun actool \
    --compile "${APP_DIR}/Resources" \
    --platform macosx \
    --minimum-deployment-target 14.0 \
    --app-icon AppIcon \
    --include-all-app-icons \
    --output-partial-info-plist "${OUT_DIR}/icon-info.plist" \
    assets/Assets.xcassets >/dev/null
rm -f "${OUT_DIR}/icon-info.plist"

cat > "${APP_DIR}/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>Scape Control</string>
    <key>CFBundleDisplayName</key>
    <string>Scape Control</string>
    <key>CFBundleIdentifier</key>
    <string>com.charlietran.scapectl</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleExecutable</key>
    <string>scapectl</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundleIconName</key>
    <string>AppIcon</string>
    <key>LSMinimumSystemVersion</key>
    <string>14.0</string>
    <key>NSHighResolutionCapable</key>
    <string>True</string>
    <key>LSUIElement</key>
    <string>1</string>
</dict>
</plist>
PLIST

# Ad-hoc sign so the bundle identifier (not "a.out") identifies the app.
codesign --force --sign - "${OUT_DIR}/ScapeCtl.app"
