DEPENDENCIES			:= cut date find git go
$(foreach dependency, ${DEPENDENCIES}, $(if $(shell which ${dependency}),, $(error No ${dependency} in PATH)))

BINARY					:= nas-cli
BUILD_DATE				:= $(shell date -u +%FT%TZ)
GIT_COMMIT				:= $(shell git rev-parse HEAD)
OUTPUT_DIR				:= build
TAG						:= $(shell git describe --abbrev=0)
NEXT_VERSION_BASE		:= $(shell date +%y.%m)
VERSION_BASE			:= $(shell echo ${TAG} | cut -c1-5)
VERSION_PATCH			:= $(shell echo ${TAG} | cut -c7-)
LDFLAGS					:= -ldflags "-X github.com/jeremiergz/nas-cli/cmd/info.BuildDate=${BUILD_DATE} -X github.com/jeremiergz/nas-cli/cmd/info.GitCommit=${GIT_COMMIT} -X github.com/jeremiergz/nas-cli/cmd/info.Version=${TAG}"
SHELL					:= /bin/bash

ifeq (${VERSION_BASE}, ${NEXT_VERSION_BASE})
	NEXT_VERSION_PATCH	:= $(shell echo $$((${VERSION_PATCH} + 1)))
else
	NEXT_VERSION_PATCH	:= 0
endif

NEXT_VERSION			:= ${NEXT_VERSION_BASE}.${NEXT_VERSION_PATCH}

define generate_binary
	@ \
	if [[ ${1} != "" ]]; then export GOOS=${1}; fi; \
	if [[ ${2} != "" ]]; then export GOARCH=${2}; fi; \
	if [[ $$GOARCH == "arm" ]]; then export GOARM=7; fi; \
	if [[ ${3} != "" ]]; then SUFFIX=-${3}; fi; \
	OUTPUT=${OUTPUT_DIR}/${BINARY}$$SUFFIX; \
	go build -mod vendor ${LDFLAGS} -o $$OUTPUT; \
	SHASUM=$$(sha256sum $$OUTPUT | awk '{print $$1}'); \
	echo "$$SHASUM  ${BINARY}$$SUFFIX" > $$OUTPUT.sha256; \
	echo ✔ successfully built [sha256: $$SHASUM] $$OUTPUT
endef

default: install

all: clean build-all install

build: clean
	@echo ➜ building v${TAG}
	$(call generate_binary,"","","")

build-all: clean
	@echo ➜ building v${TAG}
	$(call generate_binary,darwin,amd64,darwin-amd64)
	$(call generate_binary,freebsd,386,freebsd-386)
	$(call generate_binary,freebsd,amd64,freebsd-amd64)
	$(call generate_binary,linux,386,linux-386)
	$(call generate_binary,linux,amd64,linux-amd64)
	$(call generate_binary,linux,arm64,linux-arm64)
	$(call generate_binary,linux,arm,linux-armv7)
	$(call generate_binary,windows,386,windows-386.exe)
	$(call generate_binary,windows,amd64,windows-amd64.exe)

clean:
	@rm -rf ${OUTPUT_DIR}

install:
	@go install ${LDFLAGS}
	@echo ✔ successfully installed ${BINARY}

release: test
	@echo -e "\n➜ creating release v${NEXT_VERSION}"
	@git checkout main
	@git tag --annotate "${NEXT_VERSION}" --message "Release v${NEXT_VERSION}"
	@git push --follow-tags
	@echo ✔ successfully created release v${NEXT_VERSION}

test:
	@go test ./...

uninstall:
	@find "${GOPATH}/bin" -name "${BINARY}" -type f -delete
	@echo ✔ successfully uninstalled ${BINARY}
