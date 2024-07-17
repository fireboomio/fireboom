#!/usr/bin/env bash
INSERT_CHAR="/"
ASSET_FILEPATH="assets/asset.go"
if [ "$(uname -s)" == "Darwin" ];then
  sed -i '' "10s|^|${INSERT_CHAR}|" "$ASSET_FILEPATH"
  sed -i '' "13s|^|${INSERT_CHAR}|" "$ASSET_FILEPATH"
else
  sed -i "10s|^|${INSERT_CHAR}|" "$ASSET_FILEPATH"
  sed -i "13s|^|${INSERT_CHAR}|" "$ASSET_FILEPATH"
fi
