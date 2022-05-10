#!/usr/bin/env bash
set -ex

ARCHS=(
    "darwin"
    "linux"
)

REQUIRED=(
    "ipfs"
    "sha512sum"
)
for REQUIRE in "${REQUIRED[@]}"
do
    command -v "${REQUIRE}" >/dev/null 2>&1 || echo >&2 "'${REQUIRE}' must be installed"
done

mkdir bundle
pushd bundle

BINARIES=(
    "venus-miner"
)

export IPFS_PATH=`mktemp -d`
ipfs init
ipfs daemon &
PID="$!"
trap "kill -9 ${PID}" EXIT
sleep 30

for ARCH in "${ARCHS[@]}"
do
    mkdir -p "${ARCH}/venus-miner"
    pushd "${ARCH}"
    for BINARY in "${BINARIES[@]}"
    do
        cp "../../${ARCH}/${BINARY}" "venus-miner/"
        chmod +x "venus-miner/${BINARY}"
    done

    tar -zcvf "../venus-miner{CIRCLE_TAG}_${ARCH}-amd64.tar.gz" venus-miner
    popd
    rm -rf "${ARCH}"

    sha512sum "venus-miner_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz" > "venus-miner_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz.sha512"

    ipfs add -q "venus-miner_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz" > "venus-miner_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz.cid"
done
popd
