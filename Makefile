DEPENDENCIES	:= date find git go
$(foreach dependency, ${DEPENDENCIES}, $(if $(shell which ${dependency}),, $(error No ${dependency} in PATH)))

BINARY			:= nas-cli
BUILD_DATE		:= $(shell date -u +%FT%TZ)
GIT_COMMIT		:= $(shell git rev-parse HEAD)
PATCH_VERSION	?= 0
VERSION			:= $(shell date +%y.%m).$(PATCH_VERSION)
LDFLAGS			:= -ldflags "-X gitlab.com/jeremiergz/nas-cli/cmd/info.BuildDate=$(BUILD_DATE) -X gitlab.com/jeremiergz/nas-cli/cmd/info.GitCommit=$(GIT_COMMIT) -X gitlab.com/jeremiergz/nas-cli/cmd/info.Version=$(VERSION)"

default: install

all: clean build-all install

build: clean
	@go build ${LDFLAGS} -o ${BINARY}

build-all: clean
	@export GOOS=darwin;  export GOARCH=386;                   go build ${LDFLAGS} -o ${BINARY}-${VERSION}-darwin-386
	@export GOOS=darwin;  export GOARCH=amd64;                 go build ${LDFLAGS} -o ${BINARY}-${VERSION}-darwin-amd64
	@export GOOS=freebsd; export GOARCH=386;                   go build ${LDFLAGS} -o ${BINARY}-${VERSION}-freebsd-386
	@export GOOS=freebsd; export GOARCH=amd64;                 go build ${LDFLAGS} -o ${BINARY}-${VERSION}-freebsd-amd64
	@export GOOS=linux;   export GOARCH=386;                   go build ${LDFLAGS} -o ${BINARY}-${VERSION}-linux-386
	@export GOOS=linux;   export GOARCH=amd64;                 go build ${LDFLAGS} -o ${BINARY}-${VERSION}-linux-amd64
	@export GOOS=linux;   export GOARCH=arm64;                 go build ${LDFLAGS} -o ${BINARY}-${VERSION}-linux-arm64
	@export GOOS=linux;   export GOARCH=arm;   export GOARM=7; go build ${LDFLAGS} -o ${BINARY}-${VERSION}-linux-armv7
	@export GOOS=windows; export GOARCH=386;                   go build ${LDFLAGS} -o ${BINARY}-${VERSION}-windows-386.exe
	@export GOOS=windows; export GOARCH=amd64;                 go build ${LDFLAGS} -o ${BINARY}-${VERSION}-windows-amd64.exe

clean:
	@find . -name "${BINARY}*" -type f -delete

install:
	@go install ${LDFLAGS}

uninstall:
	@find "${GOPATH}/bin" -name "${BINARY}" -type f -delete
