/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/milvus-io/milvus-operator/apis/milvus.io/v1alpha1"
	scheme "github.com/milvus-io/milvus-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MilvusClustersGetter has a method to return a MilvusClusterInterface.
// A group's client should implement this interface.
type MilvusClustersGetter interface {
	MilvusClusters(namespace string) MilvusClusterInterface
}

// MilvusClusterInterface has methods to work with MilvusCluster resources.
type MilvusClusterInterface interface {
	Create(ctx context.Context, milvusCluster *v1alpha1.MilvusCluster, opts v1.CreateOptions) (*v1alpha1.MilvusCluster, error)
	Update(ctx context.Context, milvusCluster *v1alpha1.MilvusCluster, opts v1.UpdateOptions) (*v1alpha1.MilvusCluster, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.MilvusCluster, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.MilvusClusterList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MilvusCluster, err error)
	MilvusClusterExpansion
}

// milvusClusters implements MilvusClusterInterface
type milvusClusters struct {
	client rest.Interface
	ns     string
}

// newMilvusClusters returns a MilvusClusters
func newMilvusClusters(c *MilvusV1alpha1Client, namespace string) *milvusClusters {
	return &milvusClusters{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the milvusCluster, and returns the corresponding milvusCluster object, and an error if there is any.
func (c *milvusClusters) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.MilvusCluster, err error) {
	result = &v1alpha1.MilvusCluster{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("milvusclusters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MilvusClusters that match those selectors.
func (c *milvusClusters) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.MilvusClusterList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MilvusClusterList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("milvusclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested milvusClusters.
func (c *milvusClusters) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("milvusclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a milvusCluster and creates it.  Returns the server's representation of the milvusCluster, and an error, if there is any.
func (c *milvusClusters) Create(ctx context.Context, milvusCluster *v1alpha1.MilvusCluster, opts v1.CreateOptions) (result *v1alpha1.MilvusCluster, err error) {
	result = &v1alpha1.MilvusCluster{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("milvusclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(milvusCluster).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a milvusCluster and updates it. Returns the server's representation of the milvusCluster, and an error, if there is any.
func (c *milvusClusters) Update(ctx context.Context, milvusCluster *v1alpha1.MilvusCluster, opts v1.UpdateOptions) (result *v1alpha1.MilvusCluster, err error) {
	result = &v1alpha1.MilvusCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("milvusclusters").
		Name(milvusCluster.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(milvusCluster).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the milvusCluster and deletes it. Returns an error if one occurs.
func (c *milvusClusters) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("milvusclusters").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *milvusClusters) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("milvusclusters").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched milvusCluster.
func (c *milvusClusters) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.MilvusCluster, err error) {
	result = &v1alpha1.MilvusCluster{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("milvusclusters").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
