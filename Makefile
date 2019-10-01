TAGS=

.PHONY: fmt vet get clean dev setdev test

all: fmt vet test MovieNight MovieNight.exe MovieNightOSX static/main.wasm settings.json

setdev:
	$(eval export TAGS=-tags "dev")

dev: setdev all

MovieNight.exe: *.go common/*.go
	GOOS=windows GOARCH=amd64 go build -o MovieNight.exe $(TAGS)

MovieNight: *.go common/*.go
	GOOS=linux GOARCH=386 go build -o MovieNight $(TAGS)

MovieNightOSX: *.go common/*.go
	GOOS=darwin GOARCH=amd64 go build -o MovieNightOSX $(TAGS)

static/main.wasm: wasm/*.go common/*.go
	GOOS=js GOARCH=wasm go build -o ./static/main.wasm $(TAGS) wasm/*.go

clean:
	-rm MovieNight.exe MovieNight MovieNightOSX ./static/main.wasm

fmt:
	gofmt -w .

vet:
	go vet $(TAGS) ./...
	GOOS=js GOARCH=wasm go vet $(TAGS) ./...

test:
	go test $(TAGS) ./...

# Do not put settings_example.json here as a prereq to avoid overwriting
# the settings if the example is updated.
settings.json:
	cp settings_example.json settings.json
