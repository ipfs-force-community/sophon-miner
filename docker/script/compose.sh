#!/bin/sh

echo $@

token=$(cat /env/token )

echo "token:"
echo ${token}


if [ $nettype = "cali" ];then
    nettype="calibnet"
fi

echo "nettype:"
echo $nettype


    

/app/venus-miner init --nettype ${nettype} --auth-api http://127.0.0.1:8989 --token ${token} --gateway-api /ip4/127.0.0.1/tcp/45132 --api /ip4/127.0.0.1/tcp/3453 --slash-filter local

/app/venus-miner run --nettype ${nettype}
