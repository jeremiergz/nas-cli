#!/usr/bin/env bash
set -ex

BUILD_DATE="$(date -u +%FT%TZ)"
GIT_COMMIT="$(git rev-parse HEAD)"
VERSION="$(git describe --abbrev=0 --always)"
PLATFORMS=(
  darwin/amd64
  freebsd/amd64
  linux/amd64
  linux/armv7
  linux/arm64
  windows/amd64
)

for platform in "${PLATFORMS[@]}"
do
  platform_split=(${platform//\// })
  arm_split=(${platform_split[1]//v/ })
  GOOS="${platform_split[0]}"
  GOARCH="${arm_split[0]}"
  GOARM="${arm_split[1]}"
  OUTPUT="build/nas-cli-${VERSION}-${GOOS}"

  # Add .exe when OS is Windows
  if [ $GOOS = "windows" ]
  then
    OUTPUT+="-${GOARCH}.exe"

  # Add armv7 when OS is ARM
  elif [ $GOARCH = "arm" ]
  then
    OUTPUT+="-armv${GOARM}"

  # Add armv8 when OS is ARM64
  elif [ $GOARCH = "arm64" ]
  then
    OUTPUT+="-armv8"

  # Add architecture to others
  else
    OUTPUT+="-${GOARCH}"
  fi

  go build -o ${OUTPUT} -ldflags "
    -X gitlab.com/jeremiergz/nas-cli/cmd/version.BuildDate=${BUILD_DATE}
    -X gitlab.com/jeremiergz/nas-cli/cmd/version.GitCommit=${GIT_COMMIT}
    -X gitlab.com/jeremiergz/nas-cli/cmd/version.Version=${VERSION}"
done
