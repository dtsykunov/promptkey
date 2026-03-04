.PHONY: build debug check vet test

# Release binary — GUI subsystem, no console (distribution)
build:
	wails build -platform windows/amd64

# Debug binary — console window, verbose timestamped logs
debug:
	wails build -platform windows/amd64 -tags debug -o promptkey-debug.exe
	go run ./cmd/patchsubsystem build/bin/promptkey-debug.exe

# Pre-commit checks: vet + Windows cross-compile
check:
	CGO_ENABLED=0 go vet ./...
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...

vet:
	CGO_ENABLED=0 go vet ./...

test:
	go test ./...
