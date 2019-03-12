.PHONY: sync fmt vet

all: vet fmt MovieNight MovieNight.exe static/main.wasm

MovieNight.exe: *.go
	GOOS=windows GOARCH=amd64 go build -o MovieNight.exe

MovieNight: *.go
	GOOS=linux GOARCH=386 go build -o MovieNight

static/main.wasm: wasm/*.go
	GOOS=js GOARCH=wasm go build -o ./static/main.wasm wasm/*.go

clean:
	rm MovieNight.exe MovieNight ./static/main.wasm

fmt:
	gofmt -w .

vet:
	go vet ./...
	GOOS=js GOARCH=wasm go vet ./...

sync:
	scp -i /c/movienight/movienight-deploy.key -r . zorchenhimer@movienight.zorchenhimer.com:/home/zorchenhimer/movienight
