#!/bin/sh
# Wraps the macOS binary in an application bundle.
#
# macOS reads a program's name and icon from Info.plist inside a bundle and
# from nowhere else. A bare Unix binary has neither, which is what shipped
# for macOS until now: Finder drew the generic executable, the Dock drew the
# generic executable, and the window was called by the file name.
#
# Deliberately a shell script rather than lines inside the workflow: it is
# the one part of the macOS build somebody may want to run by hand on a Mac,
# and it needs nothing but sh, mkdir and cp — no Xcode, so the bundle can be
# assembled anywhere.
#
#   bundle.sh <binary> <icns> <version> <out-dir>
set -eu

binary=${1:?binary}
icns=${2:?icns}
version=${3:-0.0.0}
outdir=${4:-.}

app="$outdir/Instant Launcher.app"

# CFBundleShortVersionString is meant to be a dotted number. A build from a
# branch gets the branch name as its version, which is not one, so anything
# that does not look like a version becomes 0.0.0 rather than something
# macOS may refuse to read.
short=${version#v}
case "$short" in
    *[!0-9.]* | '' ) short=0.0.0 ;;
esac

rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"

cp "$binary" "$app/Contents/MacOS/instant-launcher"
chmod +x "$app/Contents/MacOS/instant-launcher"
cp "$icns" "$app/Contents/Resources/instant-launcher.icns"

# CFBundleIconFile carries no extension: macOS appends .icns itself, and a
# name with one in it is a common way to end up with no icon and no error.
cat > "$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>Instant Launcher</string>
	<key>CFBundleDisplayName</key>
	<string>Instant Launcher</string>
	<key>CFBundleExecutable</key>
	<string>instant-launcher</string>
	<key>CFBundleIdentifier</key>
	<string>net.geraldhofbauer.instant-launcher</string>
	<key>CFBundleIconFile</key>
	<string>instant-launcher</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleShortVersionString</key>
	<string>$short</string>
	<key>CFBundleVersion</key>
	<string>$short</string>
	<key>LSApplicationCategoryType</key>
	<string>public.app-category.games</string>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
PLIST

# PkgInfo is vestigial and costs eight bytes; some older tooling still looks
# for it and none is upset by its being there.
printf 'APPL????' > "$app/Contents/PkgInfo"

echo "  bundled $app (version $short)"
