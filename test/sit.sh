#!/bin/bash

# System Integration Test

# utils
export LOG_PATH=/tmp/sit.log
log() {
    echo "$(date +"%Y-%m-%d %H:%M:%S") $1" >> $LOG_PATH
}

# cases:
case_create_delete(){
    set -e
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
        log "$(date +"%Y-%m-%d %H:%M:%S") MilvusCluster status: $CR_STATUS"
        ATTEMPTS=$((ATTEMPTS + 1))
        sleep 10
    done

    if [ "$CR_STATUS" != "Healthy" ]; then
        log "MilvusCluster creation failed"
        log "MilvusCluster final yaml: \n $(kubectl get -n mc-sit mc/mc-sit -o yaml)"
        exit 1
    fi

    # Delete CR
    log "Deleting MilvusCluster ..." >> $LOG_PATH
    kubectl delete -f test/min-mc.yaml
}

# test case start banner
echo "==============================="
echo "System Integration Test Start"
echo "==============================="
echo "log can be found in $LOG_PATH"
echo "" > $LOG_PATH

success=0
count=0

cases=(
    case_create_delete
)

echo "Running total: ${#cases[@]} CASES"

# run each test case in sequence
for case in "${cases[@]}"; do
    echo "Running CASE[$count]: $case ..."
    $case >> $LOG_PATH
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
    echo "==============================="
    echo "Detail Logs"
    echo "==============================="
    cat $LOG_PATH
    exit 1
fi