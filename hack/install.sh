#!/bin/sh
# Copyright 2023 Greptime Team
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


set -ue

OS_TYPE=
ARCH_TYPE=
VERSION=${1:-latest}
GITHUB_ORG=${2:-GreptimeTeam}
GITHUB_REPO=gtctl
BIN=gtctl

get_os_type() {
    os_type="$(uname -s)"

    case "$os_type" in
    Darwin)
        OS_TYPE=darwin
        ;;
    Linux)
        OS_TYPE=linux
        ;;
    *)
        echo "Error: Unknown OS type: $os_type"
        exit 1
    esac
}

get_arch_type() {
    arch_type="$(uname -m)"

    case "$arch_type" in
    arm64)
        ARCH_TYPE=arm64
        ;;
    aarch64)
        ARCH_TYPE=arm64
        ;;
    x86_64)
        ARCH_TYPE=amd64
        ;;
    amd64)
        ARCH_TYPE=amd64
        ;;
    *)
        echo "Error: Unknown CPU type: $arch_type"
        exit 1
    esac
}

get_os_type
get_arch_type

if [ -n "${OS_TYPE}" ] && [ -n "${ARCH_TYPE}" ]; then
    echo "Downloading ${BIN}, OS: ${OS_TYPE}, Arch: ${ARCH_TYPE}, Version: ${VERSION}"

    if [ "${VERSION}" = "latest" ]; then
        wget "https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/latest/download/${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz"
    else
        wget "https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/${VERSION}/${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz"
    fi

    tar xvf ${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz && rm ${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz && echo "Run '${BIN} --help' to get started"
fi
