#!/usr/bin/env bash

logDir=../merged-log
if [ ! -d $logDir ]; then
  mkdir "$logDir"
fi
logFilePath=$logDir/$(date "+%Y-%m-%d").txt
cd ../wundergraphGitSubmodule || exit
echo "commit hash：$(git rev-parse HEAD)" > "$logFilePath"
# shellcheck disable=SC2129
echo "commit tag：$(git describe --tags --abbrev=0)" >> "$logFilePath"
echo "commit message：" >> "$logFilePath"
git log -1 --pretty=%B >> "$logFilePath"
