FROM filvenus/venus-buildenv AS buildenv

WORKDIR /build

COPY ./go.mod /build/
COPY ./exter[n] ./go.mod  /build/extern/
RUN export GOPROXY=https://goproxy.cn,direct && go mod download 

COPY . /build
RUN export GOPROXY=https://goproxy.cn,direct  && make



FROM filvenus/venus-runtime

ARG BUILD_TARGET=venus-miner

# DIR for app
WORKDIR /app

# copy the app from build env
COPY --from=buildenv  /build/${BUILD_TARGET} /app/${BUILD_TARGET}

COPY ./docker/script  /script


ENTRYPOINT ["/script/init.sh"]
