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
    "sophon-miner"
)

# start ipfs
export IPFS_PATH=`mktemp -d`
ipfs init
ipfs daemon &
PID="$!"
trap "kill -9 ${PID}" EXIT
sleep 30

for ARCH in "${ARCHS[@]}"
do
    mkdir -p "${ARCH}/sophon-miner"
    pushd "${ARCH}"
    for BINARY in "${BINARIES[@]}"
    do
        cp "../../${ARCH}/${BINARY}" "sophon-miner/"
        chmod +x "sophon-miner/${BINARY}"
    done

    tar -zcvf "../sophon-miner_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz" sophon-miner
    popd
    rm -rf "${ARCH}"

    sha512sum "sophon-miner_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz" > "sophon-miner_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz.sha512"

    ipfs add -q "sophon-miner_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz" > "sophon-miner_${CIRCLE_TAG}_${ARCH}-amd64.tar.gz.cid"
done
popd
