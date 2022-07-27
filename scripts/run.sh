#!/bin/bash
set -e
MilvusUserConfigMountPath="/milvus/configs/user.yaml"
MilvusOriginalConfigPath="/milvus/configs/milvus.yaml"
# merge config
/milvus/tools/merge -s ${MilvusUserConfigMountPath} -d ${MilvusOriginalConfigPath}
# run commands
exec $@
