#!/bin/bash

export GOBIN=/home/tiger/go/bin
export PATH=$PATH:$GOBIN

NAME=toutiao.microservice.tsad

mkdir -p output/bin output/conf
cp bootstrap.sh settings.py output

chmod +x  output/bootstrap.sh
cp conf/* output/conf

go build -o output/bin/${NAME}
