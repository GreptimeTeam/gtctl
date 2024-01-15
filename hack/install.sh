#!/bin/sh
# Copyright 2024 Greptime Team
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ue

OS_TYPE=
ARCH_TYPE=
VERSION="latest"
GITHUB_ORG="GreptimeTeam"
GITHUB_REPO="gtctl"
BIN="gtctl"
DOWNLOAD_SOURCE="github"
INSTALL_DIR=$(pwd)
GREPTIME_AWS_CN_RELEASE_BUCKET="https://downloads.greptime.cn/releases"

usage() {
    echo "Usage: $0 [-s <source>] [-v <version>] [-o <organization>] [-i]"
    echo "Options:"
    echo "  -s  Download source. Options: github, aws. Default: github."
    echo "  -v  Version of the binary to install. Default: latest."
    echo "  -o  Organization of the repository. Default: GreptimeTeam."
    echo "  -r  Repository name. Default: gtctl."
    echo "  -i  Install to /usr/local/bin."
}

download_from_github() {
    if [ "${VERSION}" = "latest" ]; then
        wget "https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/latest/download/${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz"
        wget "https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/latest/download/${BIN}-${OS_TYPE}-${ARCH_TYPE}.sha256sum"
    else
        wget "https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/${VERSION}/${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz"
        wget "https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/${VERSION}/${BIN}-${OS_TYPE}-${ARCH_TYPE}.sha256sum"
    fi
}

download_from_aws() {
    wget "$GREPTIME_AWS_CN_RELEASE_BUCKET/${GITHUB_REPO}/${VERSION}/${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz"
    wget "$GREPTIME_AWS_CN_RELEASE_BUCKET/${GITHUB_REPO}/${VERSION}/${BIN}-${OS_TYPE}-${ARCH_TYPE}.sha256sum"
}

verify_sha256sum() {
    command -v shasum >/dev/null 2>&1 || { echo "WARN: shasum command not found, skip sha256sum verification."; return; }

    ARTIFACT_FILE="$1"
    SUM_FILE="$2"

    # Calculate sha256sum of the downloaded file.
    CALCULATE_SUM=$(shasum -a 256 "$ARTIFACT_FILE" | cut -f1 -d' ')

    if [ "${CALCULATE_SUM}" != "$(cat "$SUM_FILE")" ]; then
        echo "ERROR: sha256sum verification failed for $ARTIFACT_FILE"
        exit 1
    else
        echo "sha256sum verification succeeded for $ARTIFACT_FILE, checksum: $CALCULATE_SUM"
    fi
}

install_binary() {
    tar xvf "${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz"
    if [ "${INSTALL_DIR}" = "/usr/local/bin/" ]; then
        sudo mv "${BIN}" "${INSTALL_DIR}"
        echo "Run '${BIN} --help' to get started"
    else
        echo "Run './${BIN} --help' to get started"
    fi

    # Clean the download package.
    rm "${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz"
    rm "${BIN}-${OS_TYPE}-${ARCH_TYPE}.sha256sum"
}

get_os_type() {
    case "$(uname -s)" in
        Darwin) OS_TYPE=darwin;;
        Linux) OS_TYPE=linux;;
        *) echo "Error: Unknown OS type"; exit 1;;
    esac
}

get_arch_type() {
    case "$(uname -m)" in
        arm64|aarch64) ARCH_TYPE=arm64;;
        x86_64|amd64) ARCH_TYPE=amd64;;
        *) echo "Error: Unknown CPU type"; exit 1;;
    esac
}

do_download() {
  echo "Downloading '${BIN}' from '${DOWNLOAD_SOURCE}', OS: '${OS_TYPE}', Arch: '${ARCH_TYPE}', Version: '${VERSION}'"

  case "${DOWNLOAD_SOURCE}" in
      github) download_from_github;;
      aws) download_from_aws;;
      *) echo "ERROR: Unknown download source"; exit 1;;
  esac
}

# Check required commands
command -v wget >/dev/null 2>&1 || { echo "ERROR: wget command not found. Please install wget."; exit 1; }

while getopts "s:v:o:r:i" opt; do
    case "$opt" in
        s) DOWNLOAD_SOURCE="$OPTARG";;
        v) VERSION="$OPTARG";;
        o) GITHUB_ORG="$OPTARG";;
        r) GITHUB_REPO="$OPTARG";;
        i) INSTALL_DIR="/usr/local/bin/";;
        *) usage; exit 1;;
    esac
done

# Main
get_os_type
get_arch_type
do_download
verify_sha256sum "$(pwd)"/${BIN}-${OS_TYPE}-${ARCH_TYPE}.tgz "$(pwd)"/${BIN}-${OS_TYPE}-${ARCH_TYPE}.sha256sum
install_binary
