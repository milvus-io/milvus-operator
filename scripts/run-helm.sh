#!/bin/bash
set -e
MilvusHelmUserConfigMountPath="/milvus/configs/user.yaml"
MilvusHelmDefaultConfigMountPath="/milvus/configs/default.yaml"
MilvusOriginalConfigPath="/milvus/configs/milvus.yaml"
# merge config
/milvus/tools/merge -s ${MilvusHelmDefaultConfigMountPath} -d ${MilvusOriginalConfigPath}
/milvus/tools/merge -s ${MilvusHelmUserConfigMountPath} -d ${MilvusOriginalConfigPath}
# run commands
exec $@
