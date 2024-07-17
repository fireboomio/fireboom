#!/usr/bin/env bash
datetime=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")
mkdir -p release
echo "$datetime" > release/build_time
export FB_COMMIT=$(git rev-parse --short HEAD)

CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.FbVersion=$FB_VERSION -X main.FbCommit=$FB_COMMIT" -o release/fireboom-mac$BIN_SUFFIX app/main.go
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.FbVersion=$FB_VERSION -X main.FbCommit=$FB_COMMIT" -o release/fireboom-mac-arm64$BIN_SUFFIX app/main.go
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.FbVersion=$FB_VERSION -X main.FbCommit=$FB_COMMIT" -o release/fireboom-linux$BIN_SUFFIX app/main.go
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.FbVersion=$FB_VERSION -X main.FbCommit=$FB_COMMIT" -o release/fireboom-linux-arm64$BIN_SUFFIX app/main.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.FbVersion=$FB_VERSION -X main.FbCommit=$FB_COMMIT" -o release/fireboom-windows$BIN_SUFFIX.exe app/main.go
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags "-s -w -X main.FbVersion=$FB_VERSION -X main.FbCommit=$FB_COMMIT" -o release/fireboom-windows-arm64$BIN_SUFFIX.exe app/main.go