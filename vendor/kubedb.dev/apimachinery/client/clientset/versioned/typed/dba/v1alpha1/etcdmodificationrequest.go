/*
Copyright The KubeDB Authors.

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
	"time"

	v1alpha1 "kubedb.dev/apimachinery/apis/dba/v1alpha1"
	scheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// EtcdModificationRequestsGetter has a method to return a EtcdModificationRequestInterface.
// A group's client should implement this interface.
type EtcdModificationRequestsGetter interface {
	EtcdModificationRequests() EtcdModificationRequestInterface
}

// EtcdModificationRequestInterface has methods to work with EtcdModificationRequest resources.
type EtcdModificationRequestInterface interface {
	Create(*v1alpha1.EtcdModificationRequest) (*v1alpha1.EtcdModificationRequest, error)
	Update(*v1alpha1.EtcdModificationRequest) (*v1alpha1.EtcdModificationRequest, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.EtcdModificationRequest, error)
	List(opts v1.ListOptions) (*v1alpha1.EtcdModificationRequestList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.EtcdModificationRequest, err error)
	EtcdModificationRequestExpansion
}

// etcdModificationRequests implements EtcdModificationRequestInterface
type etcdModificationRequests struct {
	client rest.Interface
}

// newEtcdModificationRequests returns a EtcdModificationRequests
func newEtcdModificationRequests(c *DbaV1alpha1Client) *etcdModificationRequests {
	return &etcdModificationRequests{
		client: c.RESTClient(),
	}
}

// Get takes name of the etcdModificationRequest, and returns the corresponding etcdModificationRequest object, and an error if there is any.
func (c *etcdModificationRequests) Get(name string, options v1.GetOptions) (result *v1alpha1.EtcdModificationRequest, err error) {
	result = &v1alpha1.EtcdModificationRequest{}
	err = c.client.Get().
		Resource("etcdmodificationrequests").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of EtcdModificationRequests that match those selectors.
func (c *etcdModificationRequests) List(opts v1.ListOptions) (result *v1alpha1.EtcdModificationRequestList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.EtcdModificationRequestList{}
	err = c.client.Get().
		Resource("etcdmodificationrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested etcdModificationRequests.
func (c *etcdModificationRequests) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Resource("etcdmodificationrequests").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a etcdModificationRequest and creates it.  Returns the server's representation of the etcdModificationRequest, and an error, if there is any.
func (c *etcdModificationRequests) Create(etcdModificationRequest *v1alpha1.EtcdModificationRequest) (result *v1alpha1.EtcdModificationRequest, err error) {
	result = &v1alpha1.EtcdModificationRequest{}
	err = c.client.Post().
		Resource("etcdmodificationrequests").
		Body(etcdModificationRequest).
		Do().
		Into(result)
	return
}

// Update takes the representation of a etcdModificationRequest and updates it. Returns the server's representation of the etcdModificationRequest, and an error, if there is any.
func (c *etcdModificationRequests) Update(etcdModificationRequest *v1alpha1.EtcdModificationRequest) (result *v1alpha1.EtcdModificationRequest, err error) {
	result = &v1alpha1.EtcdModificationRequest{}
	err = c.client.Put().
		Resource("etcdmodificationrequests").
		Name(etcdModificationRequest.Name).
		Body(etcdModificationRequest).
		Do().
		Into(result)
	return
}

// Delete takes name of the etcdModificationRequest and deletes it. Returns an error if one occurs.
func (c *etcdModificationRequests) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("etcdmodificationrequests").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *etcdModificationRequests) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Resource("etcdmodificationrequests").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched etcdModificationRequest.
func (c *etcdModificationRequests) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.EtcdModificationRequest, err error) {
	result = &v1alpha1.EtcdModificationRequest{}
	err = c.client.Patch(pt).
		Resource("etcdmodificationrequests").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
