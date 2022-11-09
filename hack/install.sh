#!/bin/sh

OS_TYPE=
ARCH_TYPE=

set -u

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
        echo "unknown CPU type: $os_type"
        exit 1
    esac
}

get_arch_type() {
    arch_type="$(uname -p)"
    os_type="$(uname -s)"

    case "$arch_type" in
    arm)
        if [ "$os_type" = "Darwin" ]; then
            ARCH_TYPE=arm64
        fi
        ;;
    aarch64)
        ARCH_TYPE=arm64
        ;;
    *)
        echo "unknown CPU type: $arch_type"
        exit 1
    esac
}

get_os_type
get_arch_type

if [ -n "$OS_TYPE" ] && [ -n "$ARCH_TYPE" ]; then
    wget "https://github.com/GreptimeTeam/gtctl/releases/latest/download/gtctl-$OS_TYPE-$ARCH_TYPE.tgz"
    tar xvf gtctl-$OS_TYPE-$ARCH_TYPE.tgz && rm gtctl-$OS_TYPE-$ARCH_TYPE.tgz
fi
