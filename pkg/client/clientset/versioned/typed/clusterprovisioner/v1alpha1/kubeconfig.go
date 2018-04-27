/*
Copyright 2018 The Kubernetes Authors.

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

package v1alpha1

import (
	v1alpha1 "github.com/rancher/kubecon2018/pkg/apis/clusterprovisioner/v1alpha1"
	scheme "github.com/rancher/kubecon2018/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// KubeconfigsGetter has a method to return a KubeconfigInterface.
// A group's client should implement this interface.
type KubeconfigsGetter interface {
	Kubeconfigs() KubeconfigInterface
}

// KubeconfigInterface has methods to work with Kubeconfig resources.
type KubeconfigInterface interface {
	Create(*v1alpha1.Kubeconfig) (*v1alpha1.Kubeconfig, error)
	Update(*v1alpha1.Kubeconfig) (*v1alpha1.Kubeconfig, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Kubeconfig, error)
	List(opts v1.ListOptions) (*v1alpha1.KubeconfigList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Kubeconfig, err error)
	KubeconfigExpansion
}

// kubeconfigs implements KubeconfigInterface
type kubeconfigs struct {
	client rest.Interface
}

// newKubeconfigs returns a Kubeconfigs
func newKubeconfigs(c *ClusterprovisionerV1alpha1Client) *kubeconfigs {
	return &kubeconfigs{
		client: c.RESTClient(),
	}
}

// Get takes name of the kubeconfig, and returns the corresponding kubeconfig object, and an error if there is any.
func (c *kubeconfigs) Get(name string, options v1.GetOptions) (result *v1alpha1.Kubeconfig, err error) {
	result = &v1alpha1.Kubeconfig{}
	err = c.client.Get().
		Resource("kubeconfigs").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Kubeconfigs that match those selectors.
func (c *kubeconfigs) List(opts v1.ListOptions) (result *v1alpha1.KubeconfigList, err error) {
	result = &v1alpha1.KubeconfigList{}
	err = c.client.Get().
		Resource("kubeconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested kubeconfigs.
func (c *kubeconfigs) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Resource("kubeconfigs").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a kubeconfig and creates it.  Returns the server's representation of the kubeconfig, and an error, if there is any.
func (c *kubeconfigs) Create(kubeconfig *v1alpha1.Kubeconfig) (result *v1alpha1.Kubeconfig, err error) {
	result = &v1alpha1.Kubeconfig{}
	err = c.client.Post().
		Resource("kubeconfigs").
		Body(kubeconfig).
		Do().
		Into(result)
	return
}

// Update takes the representation of a kubeconfig and updates it. Returns the server's representation of the kubeconfig, and an error, if there is any.
func (c *kubeconfigs) Update(kubeconfig *v1alpha1.Kubeconfig) (result *v1alpha1.Kubeconfig, err error) {
	result = &v1alpha1.Kubeconfig{}
	err = c.client.Put().
		Resource("kubeconfigs").
		Name(kubeconfig.Name).
		Body(kubeconfig).
		Do().
		Into(result)
	return
}

// Delete takes name of the kubeconfig and deletes it. Returns an error if one occurs.
func (c *kubeconfigs) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Resource("kubeconfigs").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *kubeconfigs) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Resource("kubeconfigs").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched kubeconfig.
func (c *kubeconfigs) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Kubeconfig, err error) {
	result = &v1alpha1.Kubeconfig{}
	err = c.client.Patch(pt).
		Resource("kubeconfigs").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
