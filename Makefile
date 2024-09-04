define check_deps
	$(foreach dependency, ${1}, $(if $(shell which ${dependency}),, $(error No ${dependency} in PATH)))
endef

$(call check_deps,basename cut date echo git go)

DEPENDENCIES			:= awk cut date echo git go rm sha256sum
$(foreach dependency, ${DEPENDENCIES}, $(if $(shell which ${dependency}),, $(error No ${dependency} in PATH)))

TAG						:= $(shell git describe --abbrev=0 2>/dev/null)
ifeq (${TAG},)
	TAG					:= N/A
endif

OUTPUT_DIR				:= build
COVERAGE_DIR			:= coverage
MODULE					:= $(shell go list -m)
BINARY					:= $(shell basename ${MODULE})
SHELL					:= /bin/bash

BUILD_DATE				:= $(shell date -u +%FT%T.%3NZ)
GIT_COMMIT				:= $(shell git rev-parse HEAD)

NEXT_VERSION_BASE		:= $(shell date +%y.%m)
VERSION_BASE			:= $(shell echo ${TAG} | cut -c2-6)
VERSION_PATCH			:= $(shell echo ${TAG} | cut -c8-)

ifeq (${VERSION_BASE}, ${NEXT_VERSION_BASE})
	NEXT_VERSION_PATCH	:= $(shell echo $$((${VERSION_PATCH} + 1)))
else
	NEXT_VERSION_PATCH	:= 0
endif

NEXT_VERSION			:= v${NEXT_VERSION_BASE}.${NEXT_VERSION_PATCH}

LDFLAGS					:= -ldflags
LDFLAGS					+= "
LDFLAGS					+= -X '${MODULE}/internal/config.AppName=${BINARY}'
LDFLAGS					+= -X '${MODULE}/internal/config.BuildDate=${BUILD_DATE}'
LDFLAGS					+= -X '${MODULE}/internal/config.GitCommit=${GIT_COMMIT}'
LDFLAGS					+= -X '${MODULE}/internal/config.Version=${TAG}'
LDFLAGS					+= "

define generate_binary
	@ \
	set -e; \
	if [[ ${1} != "" ]]; then export GOOS=${1}; fi; \
	if [[ ${2} != "" ]]; then export GOARCH=${2}; fi; \
	if [[ $$GOARCH == "arm" ]]; then export GOARM=7; fi; \
	if [[ ${3} != "" ]]; then SUFFIX=-${3}; fi; \
	OUTPUT=${OUTPUT_DIR}/${BINARY}$$SUFFIX; \
	go build ${LDFLAGS} -o $$OUTPUT; \
	SHASUM=$$(sha256sum $$OUTPUT | awk '{print $$1}'); \
	echo "$$SHASUM  ${BINARY}$$SUFFIX" > $$OUTPUT.sha256; \
	echo ✔ successfully built [sha256: $$SHASUM] $$OUTPUT
endef

define run_tests
	@ \
	set -e; \
	mkdir -p ${COVERAGE_DIR}; \
	go test -coverpkg=${MODULE}/... -coverprofile=${COVERAGE_DIR}/profile.cov ./...
endef

.PHONY: default
default: build

.PHONY: build
build: clean
	@echo ➜ building ${TAG}
	$(call generate_binary,"","","")

.PHONY: build-all
build-all: clean
	@echo ➜ building ${TAG}
	$(call generate_binary,darwin,amd64,darwin-amd64)
	$(call generate_binary,darwin,arm64,darwin-arm64)
	$(call generate_binary,freebsd,386,freebsd-386)
	$(call generate_binary,freebsd,amd64,freebsd-amd64)
	$(call generate_binary,linux,386,linux-386)
	$(call generate_binary,linux,amd64,linux-amd64)
	$(call generate_binary,linux,arm64,linux-arm64)
	$(call generate_binary,linux,arm,linux-armv7)
	$(call generate_binary,windows,386,windows-386.exe)
	$(call generate_binary,windows,amd64,windows-amd64.exe)

.PHONY: clean
clean:
	$(call check_deps,rm)
	@rm -rf ${OUTPUT_DIR} ${COVERAGE_DIR}

.PHONY: coverage
coverage: clean
	$(call check_deps,go)
	$(call run_tests) > /dev/null
	@go tool cover -func coverage/profile.cov

.PHONY: coverage-html
coverage-html: clean
	$(call check_deps,go)
	$(call run_tests) > /dev/null
	@go tool cover -html coverage/profile.cov

.PHONY: release
release: build test
	$(call check_deps,echo git)
	@echo -e "\n➜ creating release ${NEXT_VERSION}"
	@git checkout main
	@git tag --annotate "${NEXT_VERSION}" --message "Release ${NEXT_VERSION}"
	@git push --follow-tags
	@echo ✔ successfully created release ${NEXT_VERSION}

.PHONY: test
test: clean
	$(call check_deps,go mkdir)
	$(call run_tests)
