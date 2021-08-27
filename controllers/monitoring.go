package controllers

import (
	"context"
	"time"

	"github.com/milvus-io/milvus-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//
func (r *MilvusClusterReconciler) ConditionsCheck(ctx context.Context) {
	// do an initial check, then start the periodic check
	if err := r.checkConditions(ctx); err != nil {
		r.logger.Error(err, "conditionsCheck")
	}

	ticker := time.NewTicker(3 * time.Minute)
	for {
		select {
		case <-ticker.C:
			if err := r.checkConditions(ctx); err != nil {
				r.logger.Error(err, "conditionsCheck")
			}

		case <-ctx.Done():
			return
		}
	}
}

func (r *MilvusClusterReconciler) checkConditions(ctx context.Context) error {
	milvusclusters := &v1alpha1.MilvusClusterList{}
	if err := r.List(ctx, milvusclusters, &client.ListOptions{}); err != nil {
		return err
	}

	return nil
}
