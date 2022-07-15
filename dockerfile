FROM filvenus/venus-buildenv AS buildenv

COPY . ./venus-miner
RUN export GOPROXY=https://goproxy.cn && cd venus-miner  && make


RUN cd venus-miner && pwd && ldd ./venus-miner


FROM filvenus/venus-runtime

# DIR for app
WORKDIR /app

# copy the app from build env
COPY --from=buildenv  /go/venus-miner/venus-miner /app/venus-miner




COPY ./docker/script  /script


ENTRYPOINT ["/script/init.sh"]
