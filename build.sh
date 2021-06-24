#!/usr/bin/env bash

set -x # output detail

readonly GO_PACKAGE="github.com/imroc/zk2etcd"
ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd -P)
BIN_DIR="${ROOT_DIR}/bin"
BUILD_ROOT=${ROOT_DIR}/cmd/zk2etcd
GIT_COMMIT=$(git rev-parse "HEAD^{commit}" 2>/dev/null)
GIT_TAG="$(git describe --tags --abbrev=0 --exact-match 2>/dev/null)"
GIT_TREE_STATE=$(test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

if [ "${VERSION}" != "" ];then
    BINARY_VERSION=${VERSION}
elif [ "${GIT_TAG}" != "" ]; then
    BINARY_VERSION=${GIT_TAG}
else
    BINARY_VERSION="$(git describe --tags --abbrev=0 "${GIT_COMMIT}^{commit}" 2>/dev/null)"
    if [ "${BINARY_VERSION}" == "" ]; then
        BINARY_VERSION="v0.0.0"
    fi
fi

cls::version::ldflag() {
  local key=${1}
  local val=${2}

  echo "-X '${GO_PACKAGE}/pkg/version.${key}=${val}'"
}

cls::version::ldflags() {
#  local buildDate="--date=@$(git show -s --format=format:%ct HEAD)"

  local -a ldflags=($(cls::version::ldflag "buildDate" "$(date -u +'%Y-%m-%dT%H:%M:%SZ')"))
  ldflags+=($(cls::version::ldflag "gitCommit" "${GIT_COMMIT}"))
  ldflags+=($(cls::version::ldflag "version" "${BINARY_VERSION}"))
  ldflags+=($(cls::version::ldflag "gitTreeState" "${GIT_TREE_STATE}"))

  if [ "${GIT_TAG}" != "" ]; then # Clear the "unreleased" string in buildMetadata
    ldflags+=($(cls::version::ldflag "buildMetadata" ""))
  fi

  # The -ldflags parameter takes a single string, so join the output.
  echo "${ldflags[*]-}"
}

goldflags="-s -w $(cls::version::ldflags)"

go build -v -o ${BIN_DIR}/ -ldflags "${goldflags}" ${BUILD_ROOT}
