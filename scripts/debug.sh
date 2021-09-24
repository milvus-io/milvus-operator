#! /bin/sh

export GOPROXY=https://goproxy.cn

dlv debug --headless --log --listen :31666 --api-version 2 --accept-multiclient