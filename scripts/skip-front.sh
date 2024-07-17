#!/usr/bin/env bash
INSERT_CHAR="/"
ASSET_FILEPATH="assets/asset.go"
sed -i '' "10s|^|${INSERT_CHAR}|" "$ASSET_FILEPATH"
sed -i '' "13s|^|${INSERT_CHAR}|" "$ASSET_FILEPATH"