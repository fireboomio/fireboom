#!/usr/bin/env bash
export FB_COMMIT=$(git rev-parse --short HEAD)
go build -ldflags "-s -w -X main.FbVersion=$FB_VERSION -X main.FbCommit=$FB_COMMIT" -o release/fireboom app/main.go