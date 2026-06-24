GIT_HEAD = $(shell git rev-parse HEAD | head -c8)

build:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -gcflags "all=-trimpath=$(pwd)" -o build/photon-daemon_linux_amd64 -v main.go
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -gcflags "all=-trimpath=$(pwd)" -o build/photon-daemon_linux_arm64 -v main.go

test:
	go test -race ./...

debug:
	go build -ldflags="-X github.com/Photon-Panel/Photon-Daemon/system.Version=$(GIT_HEAD)"
	sudo ./photon-daemon --debug --ignore-certificate-errors --config config.yml --pprof --pprof-block-rate 1

# Runs a remotly debuggable session for the daemon allowing an IDE to connect and target
# different breakpoints.
rmdebug:
	go build -gcflags "all=-N -l" -ldflags="-X github.com/Photon-Panel/Photon-Daemon/system.Version=$(GIT_HEAD)" -race
	sudo dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./photon-daemon -- --debug --ignore-certificate-errors --config config.yml

cross-build: clean build compress

clean:
	rm -rf build/photon-daemon_*

.PHONY: all build compress clean
