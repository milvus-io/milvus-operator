package controllers

import (
	"fmt"

	milvusiov1alpha1 "github.com/milvus-io/milvus-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var debugLog = logf.Log.WithName("predicates").WithName("Debug")

type debugPredicate struct{}

func DebugPredicate() *debugPredicate {
	return &debugPredicate{}
}

var _ predicate.Predicate = &debugPredicate{}

// Create returns true if the Create event should be processed
func (*debugPredicate) Create(evt event.CreateEvent) bool {
	/* mc, ok := evt.Object.(*milvusiov1alpha1.MilvusCluster)
	if ok {
		debugLog.Info("Create", "mc", mc)
	} */
	debugLog.Info("Create", "object", evt.Object)
	return true
}

// Delete returns true if the Delete event should be processed
func (*debugPredicate) Delete(_ event.DeleteEvent) bool {
	return true
}

// Update returns true if the Update event should be processed
func (*debugPredicate) Update(evt event.UpdateEvent) bool {
	_, ok := evt.ObjectOld.(*milvusiov1alpha1.MilvusCluster)
	if ok {
		obj := fmt.Sprintf("%s/%s", evt.ObjectNew.GetNamespace(), evt.ObjectNew.GetName())
		diff, err := client.MergeFrom(evt.ObjectOld).Data(evt.ObjectNew)
		if err != nil {
			debugLog.Info("error generating diff", "err", err, "obj", obj)
		} else {
			debugLog.Info("Update diff", "diff", string(diff), "obj", obj)
		}
		debugLog.Info("IsEqual", "equal", IsEqual(evt.ObjectOld, evt.ObjectNew), "obj", obj)
	}

	return true
}

// Generic returns true if the Generic event should be processed
func (*debugPredicate) Generic(_ event.GenericEvent) bool {
	return true
}
