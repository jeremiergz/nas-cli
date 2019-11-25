APP_NAME		:= $(shell basename "$(PWD)")
BUILD_DATE		:= $(shell date -u +%FT%TZ)
BUILD_DIR		:= build
GIT_COMMIT		:= $(shell git rev-parse HEAD)
PATCH_VERSION	?= 0
VERSION			:= $(shell date +%y.%m).$(PATCH_VERSION)
LDFLAGS			:= -X gitlab.com/jeremiergz/nas-cli/cmd/info.BuildDate=$(BUILD_DATE) -X gitlab.com/jeremiergz/nas-cli/cmd/info.GitCommit=$(GIT_COMMIT) -X gitlab.com/jeremiergz/nas-cli/cmd/info.Version=$(VERSION)

install:
	@go install -ldflags "$(LDFLAGS)"

build: clean compile

build-all: clean compile-all

clean:
	@find . -name "$(APP_NAME)*" -type f -delete

compile:
	@go build -ldflags "$(LDFLAGS)"

compile-all:
	@goos=darwin  GOARCH=amd64         go build -o "$(APP_NAME)-$(VERSION)-darwin-amd64"      -ldflags "$(LDFLAGS)"
	@GOOS=freebsd GOARCH=amd64         go build -o "$(APP_NAME)-$(VERSION)-freebsd-amd64"     -ldflags "$(LDFLAGS)"
	@GOOS=linux   GOARCH=amd64         go build -o "$(APP_NAME)-$(VERSION)-linux-amd64"       -ldflags "$(LDFLAGS)"
	@GOOS=linux   GOARCH=arm64         go build -o "$(APP_NAME)-$(VERSION)-linux-arm64"       -ldflags "$(LDFLAGS)"
	@GOOS=linux   GOARCH=arm   GOARM=7 go build -o "$(APP_NAME)-$(VERSION)-linux-armv7"       -ldflags "$(LDFLAGS)"
	@GOOS=windows GOARCH=amd64         go build -o "$(APP_NAME)-$(VERSION)-windows-amd64.exe" -ldflags "$(LDFLAGS)"

uninstall:
	@find "$(GOPATH)/bin" -name "$(APP_NAME)" -type f -delete
