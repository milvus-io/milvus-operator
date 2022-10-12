#!/bin/bash
set -ex
echo "Deploying old milvus"
kubectl apply -f config/samples/demo.yaml
kubectl --timeout 10m wait --for=condition=MilvusReady mi my-release
echo "Deploying milvus upgrade"
kubectl apply -f config/samples/beta/milvusupgrade.yaml
kubectl --timeout 10m wait --for=condition=Upgraded milvusupgrade my-release-upgrade
kubectl get mi -o yaml
echo "Rollback"
kubectl patch milvusupgrade my-release-upgrade --patch '{"spec": {"operation": "rollback"}}' --type=merge
kubectl --timeout 10m wait --for=condition=Rollbacked milvusupgrade my-release-upgrade
kubectl get mi -o yaml
echo "Clean up"
kubectl delete -f config/samples/demo.yaml --wait=true --timeout=5m --cascade=foreground
kubectl delete -f config/samples/beta/milvusupgrade.yaml --wait=true --timeout=5m --cascade=foreground

sleep 10

echo "Deploying old milvus cluster"
kubectl apply -f config/samples/cluster_demo.yaml
kubectl --timeout 15m wait --for=condition=MilvusReady mi my-release
kubectl get mi -o yaml
echo "Deploying milvus upgrade"
kubectl apply -f config/samples/beta/milvusupgrade.yaml
kubectl --timeout 10m wait --for=condition=Upgraded milvusupgrade my-release-upgrade
echo "Rollback"
kubectl patch milvusupgrade my-release-upgrade --patch '{"spec": {"operation": "rollback"}}' --type=merge
kubectl --timeout 10m wait --for=condition=Rollbacked milvusupgrade my-release-upgrade
kubectl get mi -o yaml
