cask "scapectl" do
  version "0.0.14"
  sha256 :no_check

  url "https://github.com/charlietran/scapectl/releases/download/v#{version}/Mac_ScapeCtl.zip"
  name "Scape Control"
  desc "System tray app + CLI for the Fractal Design Scape wireless headset"
  homepage "https://github.com/charlietran/scapectl"

  livecheck do
    url :url
    strategy :github_latest
  end

  depends_on :macos

  # Installs the .app into /Applications and symlinks the inner binary
  # into Homebrew's bin so `scapectl` still works on the command line.
  app "ScapeCtl.app"
  binary "#{appdir}/ScapeCtl.app/Contents/MacOS/scapectl"

  # The release binary is ad-hoc signed by the linker but not notarized.
  # macOS Sequoia tracks both com.apple.quarantine and com.apple.provenance,
  # and Gatekeeper rejects "downloaded ad-hoc" signatures with the
  # "ScapeCtl.app is damaged" dialog (which auto-moves it to Trash).
  # Strip all xattrs and re-sign locally — a fresh local ad-hoc signature
  # is trusted by Gatekeeper where the downloaded one is not.
  postflight_steps do
    run "/usr/bin/xattr",
        args:         ["-cr", "{{appdir}}/ScapeCtl.app"],
        must_succeed: false
    run "/usr/bin/codesign",
        args:         ["--force", "--deep", "--sign", "-", "{{appdir}}/ScapeCtl.app"],
        must_succeed: false
  end

  zap trash: [
    "~/.config/scapectl",
    "~/Library/Application Support/scapectl",
  ]
end
