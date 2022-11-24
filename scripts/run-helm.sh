#!/bin/bash
set -e
MilvusHelmUserConfigMountPath="/milvus/configs/user.yaml"
MilvusHelmDefaultConfigMountPath="/milvus/configs/default.yaml"
MilvusOriginalConfigPath="/milvus/configs/milvus.yaml"
# merge config
/milvus/tools/merge -s ${MilvusHelmDefaultConfigMountPath} -d ${MilvusOriginalConfigPath}
/milvus/tools/merge -s ${MilvusHelmUserConfigMountPath} -d ${MilvusOriginalConfigPath}
echo "helm default config:"
cat /milvus/configs/default.yaml
echo
echo "helm user config:"
cat /milvus/configs/user.yaml
echo
echo "config after merging:"
cat /milvus/configs/milvus.yaml
echo
# run commands
exec $@
