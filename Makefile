DEPENDENCIES	:= cut date find git go
$(foreach dependency, ${DEPENDENCIES}, $(if $(shell which ${dependency}),, $(error No ${dependency} in PATH)))

BINARY				:= nas-cli
BUILD_DATE			:= $(shell date -u +%FT%TZ)
GIT_COMMIT			:= $(shell git rev-parse HEAD)
PATCH_VERSION		?= 0
TAG					:= $(shell git describe --abbrev=0)
PREV_VERSION_BASE	:= $(shell echo ${TAG} | cut -c1-5)
PREV_VERSION_PATCH	:= $(shell echo ${TAG} | cut -c7-7)
NEXT_VERSION_BASE	:= $(shell date +%y.%m)
VERSION				:= $(shell date +%y.%m).${PATCH_VERSION}
LDFLAGS				:= -ldflags "-X gitlab.com/jeremiergz/nas-cli/cmd/info.BuildDate=${BUILD_DATE} -X gitlab.com/jeremiergz/nas-cli/cmd/info.GitCommit=${GIT_COMMIT} -X gitlab.com/jeremiergz/nas-cli/cmd/info.Version=${VERSION}"

ifeq (${PREV_VERSION_BASE}, ${NEXT_VERSION_BASE})
	NEXT_VERSION_PATCH	:= $(shell echo $$((PREV_VERSION_PATCH + 1)))
	VERSION				:= ${NEXT_VERSION_BASE}.${NEXT_VERSION_PATCH}
else
	NEXT_VERSION_PATCH	:= 0
	VERSION				:= ${NEXT_VERSION_BASE}.${NEXT_VERSION_PATCH}
endif

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

release:
	@git checkout master
	@git rebase develop
	@git tag --annotate "${VERSION}" --message "Release v${VERSION}"
	@git push --follow-tags
	@git checkout develop
	@git rebase master
	@git push

uninstall:
	@find "${GOPATH}/bin" -name "${BINARY}" -type f -delete
