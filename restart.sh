#!/usr/bin/env bash

# kill 历史进程
for pid in $(ps -ef | grep fireboom| grep -v grep | awk '{print $2}'); do
kill -9 $pid
sleep 1
done
# kill prisma引擎
pkill prisma*

# 重启
nohup ./fireboom &



# 默认使用goproxy.cn
export GOPROXY=https://goproxy.cn
# input your command here
go install github.com/swaggo/swag/cmd/swag@latest

swag init -g app/main.go -o app/docs

CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o release/fireboom-darwin app/main.go
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o release/fireboom-linux app/main.go
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o release/fireboom-linux-arm64 app/main.go
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o release/fireboom-windows.exe app/main.go


