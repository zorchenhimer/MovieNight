.PHONY: fmt vet get clean

all: fmt vet MovieNight MovieNight.exe static/main.wasm

MovieNight.exe: *.go common/*.go
	GOOS=windows GOARCH=amd64 go build -o MovieNight.exe

MovieNight: *.go common/*.go
	GOOS=linux GOARCH=386 go build -o MovieNight

static/main.wasm: wasm/*.go common/*.go
	GOOS=js GOARCH=wasm go build -o ./static/main.wasm wasm/*.go

clean:
	-rm MovieNight.exe MovieNight ./static/main.wasm

fmt:
	goimports -w .

get:
	go get -u -v ./...
	GOOS=js GOARCH=wasm go get -u -v ./...
	go get golang.org/x/tools/cmd/goimports

vet:
	go vet ./...
	GOOS=js GOARCH=wasm go vet ./...
