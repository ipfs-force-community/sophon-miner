#!/bin/sh

echo "EXEC: ./venus-miner $@ ; ./venus-miner run \n\n"
./venus-miner $@
./venus-miner run
