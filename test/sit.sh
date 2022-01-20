#!/bin/bash

# System Integration Test

# utils
export LOG_PATH=/tmp/sit.log
log() {
    echo "$(date +"%Y-%m-%d %H:%M:%S") $1"
}

# milvus cluster cases:
case_create_delete_cluster(){
    # create MilvusCluster CR
    log "Creating MilvusCluster..."
    kubectl apply -f test/min-mc.yaml

    # Check CR status every 10 seconds (max 10 minutes) until complete.
    ATTEMPTS=0
    CR_STATUS=""
    until [ $ATTEMPTS -eq 60 ]; 
    do
        CR_STATUS=$(kubectl get -n mc-sit mc/mc-sit -o=jsonpath='{.status.status}')
        if [ "$CR_STATUS" = "Healthy" ]; then
            break
        fi
        log "MilvusCluster status: $CR_STATUS"
        ATTEMPTS=$((ATTEMPTS + 1))
        sleep 10
    done

    if [ "$CR_STATUS" != "Healthy" ]; then
        log "MilvusCluster creation failed"
        log "MilvusCluster final yaml: \n $(kubectl get -n mc-sit mc/mc-sit -o yaml)"
        log "MilvusCluster helm values: \n $(helm -n mc-sit get values mc-sit-pulsar)"
        log "MilvusCluster describe pods: \n $(kubectl -n mc-sit describe pods)"
        return 1
    fi

    # Delete CR
    log "Deleting MilvusCluster ..."
    kubectl delete -f test/min-mc.yaml
}


# milvus cases:
case_create_delete_milvus(){
    # create Milvus CR
    log "Creating Milvus..."
    kubectl apply -f test/min-milvus.yaml

    # Check CR status every 10 seconds (max 10 minutes) until complete.
    ATTEMPTS=0
    CR_STATUS=""
    until [ $ATTEMPTS -eq 60 ]; 
    do
        CR_STATUS=$(kubectl get -n milvus-sit milvus/milvus-sit -o=jsonpath='{.status.status}')
        if [ "$CR_STATUS" = "Healthy" ]; then
            break
        fi
        log "Milvus status: $CR_STATUS"
        ATTEMPTS=$((ATTEMPTS + 1))
        sleep 10
    done

    if [ "$CR_STATUS" != "Healthy" ]; then
        log "Milvus creation failed"
        log "Milvus final yaml: \n $(kubectl get -n milvus-sit milvus/milvus-sit -o yaml)"
        log "Milvus helm values: \n $(helm -n milvus-sit get values milvus-sit-pulsar)"
        log "Milvus describe pods: \n $(kubectl -n mc-sit describe pods)"
        return 1
    fi

    # Delete CR
    log "Deleting Milvus ..."
    kubectl delete -f test/min-milvus.yaml
}

success=0
count=0

cases=(
    case_create_delete_cluster
    case_create_delete_milvus
)

echo "Running total: ${#cases[@]} CASES"

# run each test case in sequence
for case in "${cases[@]}"; do
    echo "Running CASE[$count]: $case ..."
    $case
    if [ $? -eq 0 ]; then
        echo "$case [success]"
        success=$((success + 1))
    else
        echo "$case [failed]"
    fi
    count=$((count + 1))
done

# test end banner
echo "==============================="
echo "Test End"
echo "==============================="

if [ $success -eq $count ]; then
    echo "All $count tests passed"
    exit 0
else
    echo "$success of $count tests passed"
    exit 1
fi
