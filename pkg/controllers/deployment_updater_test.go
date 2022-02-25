package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestMilvus_UpdateDeployment(t *testing.T) {
	env := newMilvusTestEnv(t)
	defer env.tearDown()
	t.Run("set controllerRef failed", func(t *testing.T) {
		updater := newMilvusDeploymentUpdater(env.Inst, env.Reconciler.Scheme)
		deployment := &appsv1.Deployment{}
		err := updateDeployment(deployment, updater)
		assert.Error(t, err)
	})

	t.Run("persistence disabled", func(t *testing.T) {
		env.Inst.Spec.Persistence.Enabled = false
		updater := newMilvusDeploymentUpdater(env.Inst, env.Reconciler.Scheme)
		deployment := &appsv1.Deployment{}
		deployment.Name = "deploy"
		deployment.Namespace = "ns"
		err := updateDeployment(deployment, updater)
		assert.NoError(t, err)
		assert.Len(t, deployment.Spec.Template.Spec.Volumes, 2)
		assert.Len(t, deployment.Spec.Template.Spec.Containers[0].VolumeMounts, 2)
	})

	t.Run("persistence enabled", func(t *testing.T) {
		env.Inst.Spec.Persistence.Enabled = true
		updater := newMilvusDeploymentUpdater(env.Inst, env.Reconciler.Scheme)
		deployment := &appsv1.Deployment{}
		deployment.Name = "deploy"
		deployment.Namespace = "ns"
		err := updateDeployment(deployment, updater)
		assert.NoError(t, err)
		assert.Len(t, deployment.Spec.Template.Spec.Volumes, 3)
		assert.Len(t, deployment.Spec.Template.Spec.Containers[0].VolumeMounts, 3)
	})

	t.Run("persistence enabled using existed", func(t *testing.T) {
		env.Inst.Spec.Persistence.Enabled = true
		env.Inst.Spec.Persistence.PersistentVolumeClaim.ExistingClaim = "pvc1"
		updater := newMilvusDeploymentUpdater(env.Inst, env.Reconciler.Scheme)
		deployment := &appsv1.Deployment{}
		deployment.Name = "deploy"
		deployment.Namespace = "ns"
		err := updateDeployment(deployment, updater)
		assert.NoError(t, err)
		assert.Len(t, deployment.Spec.Template.Spec.Volumes, 3)
		assert.Equal(t, deployment.Spec.Template.Spec.Volumes[2].PersistentVolumeClaim.ClaimName, "pvc1")
		assert.Len(t, deployment.Spec.Template.Spec.Containers[0].VolumeMounts, 3)
	})
}
