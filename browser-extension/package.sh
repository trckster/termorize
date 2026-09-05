#!/bin/sh

set -eu

extension_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
version=$(node -e "const manifest = require('$extension_dir/manifest.json'); process.stdout.write(manifest.version)")
output=${1:-"$extension_dir/dist/termoclip-$version.zip"}

mkdir -p "$(dirname -- "$output")"

(
    cd "$extension_dir"
    zip -q -FS "$output" \
        manifest.json \
        background.js \
        content.js \
        content.css \
        popup.html \
        popup.css \
        popup.js \
        selection-overlay.js \
        icons/icon-16.png \
        icons/icon-32.png \
        icons/icon-48.png \
        icons/icon-128.png
)

printf 'Created %s\n' "$output"
