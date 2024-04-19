build:
	go build -o fireboom app/main.go 

build-linux:
 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o firboom_linux app/main.go

build-mac:
 CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o firboom_mac app/main.go

run:
	./fireboom dev

build-run:
	make build && make run

swagger:
	swag init -g app/main.go -o app/docs

check:
	go vet ./...