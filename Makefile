DEPENDENCIES		:= cut date find git go
$(foreach dependency, ${DEPENDENCIES}, $(if $(shell which ${dependency}),, $(error No ${dependency} in PATH)))

BINARY				:= nas-cli
BUILD_DATE			:= $(shell date -u +%FT%TZ)
GIT_COMMIT			:= $(shell git rev-parse HEAD)
PATCH_VERSION		?= 0
TAG					:= $(shell git describe --abbrev=0)
PREV_VERSION_BASE	:= $(shell echo ${TAG} | cut -c1-5)
PREV_VERSION_PATCH	:= $(shell echo ${TAG} | cut -c7-7)
NEXT_VERSION_BASE	:= $(shell date +%y.%m)
LDFLAGS				:= -ldflags "-X gitlab.com/jeremiergz/nas-cli/cmd/info.BuildDate=${BUILD_DATE} -X gitlab.com/jeremiergz/nas-cli/cmd/info.GitCommit=${GIT_COMMIT} -X gitlab.com/jeremiergz/nas-cli/cmd/info.Version=${TAG}"

ifeq (${PREV_VERSION_BASE}, ${NEXT_VERSION_BASE})
	NEXT_VERSION_PATCH	:= $(shell echo $$((${PREV_VERSION_PATCH} + 1)))
else
	NEXT_VERSION_PATCH	:= 0
endif

NEXT_VERSION		:= ${NEXT_VERSION_BASE}.${NEXT_VERSION_PATCH}

default: install

all: clean build-all install

build: clean
	@go build ${LDFLAGS} -o ${BINARY}

build-all: clean
	@export GOOS=darwin;  export GOARCH=386;                   go build ${LDFLAGS} -o ${BINARY}-darwin-386
	@export GOOS=darwin;  export GOARCH=amd64;                 go build ${LDFLAGS} -o ${BINARY}-darwin-amd64
	@export GOOS=freebsd; export GOARCH=386;                   go build ${LDFLAGS} -o ${BINARY}-freebsd-386
	@export GOOS=freebsd; export GOARCH=amd64;                 go build ${LDFLAGS} -o ${BINARY}-freebsd-amd64
	@export GOOS=linux;   export GOARCH=386;                   go build ${LDFLAGS} -o ${BINARY}-linux-386
	@export GOOS=linux;   export GOARCH=amd64;                 go build ${LDFLAGS} -o ${BINARY}-linux-amd64
	@export GOOS=linux;   export GOARCH=arm64;                 go build ${LDFLAGS} -o ${BINARY}-linux-arm64
	@export GOOS=linux;   export GOARCH=arm;   export GOARM=7; go build ${LDFLAGS} -o ${BINARY}-linux-armv7
	@export GOOS=windows; export GOARCH=386;                   go build ${LDFLAGS} -o ${BINARY}-windows-386.exe
	@export GOOS=windows; export GOARCH=amd64;                 go build ${LDFLAGS} -o ${BINARY}-windows-amd64.exe

clean:
	@find . -name "${BINARY}*" -type f -delete

install:
	@go install ${LDFLAGS}

release:
	@git checkout master
	@git rebase develop
	@git tag --annotate "${NEXT_VERSION}" --message "Release v${NEXT_VERSION}"
	@git push --follow-tags
	@git checkout develop
	@git rebase master
	@git push

uninstall:
	@find "${GOPATH}/bin" -name "${BINARY}" -type f -delete
