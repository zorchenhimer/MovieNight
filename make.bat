set GOOS=wasm
set GOARCH=js
go build -o static\main.wsm wasm\main.go wasm\suggestions.go
set GOOS=
set GOARCH=
go build -o MovieNight.exe
