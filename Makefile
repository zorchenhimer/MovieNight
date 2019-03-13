.PHONY: sync fmt vet

all: vet fmt MovieNight MovieNight.exe static/main.wasm

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

vet:
	go vet ./...
	GOOS=js GOARCH=wasm go vet ./...

sync:
	scp -i /c/movienight/movienight-deploy.key -r . zorchenhimer@movienight.zorchenhimer.com:/home/zorchenhimer/movienight

run: all
	./MovieNight.exe
