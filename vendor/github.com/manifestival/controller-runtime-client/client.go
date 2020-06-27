package client

import (
	"context"

	mf "github.com/manifestival/manifestival"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewManifest(pathname string, client client.Client, opts ...mf.Option) (mf.Manifest, error) {
	return mf.NewManifest(pathname, append(opts, mf.UseClient(NewClient(client)))...)
}

func NewClient(client client.Client) mf.Client {
	return &controllerRuntimeClient{client: client}
}

type controllerRuntimeClient struct {
	client client.Client
}

// verify implementation
var _ mf.Client = (*controllerRuntimeClient)(nil)

func (c *controllerRuntimeClient) Create(obj *unstructured.Unstructured, options *metav1.CreateOptions) error {
	return c.client.Create(context.TODO(), obj, createWith(options)...)
}

func (c *controllerRuntimeClient) Update(obj *unstructured.Unstructured, options *metav1.UpdateOptions) error {
	return c.client.Update(context.TODO(), obj, updateWith(options)...)
}

func (c *controllerRuntimeClient) Delete(obj *unstructured.Unstructured, options *metav1.DeleteOptions) error {
	return c.client.Delete(context.TODO(), obj, deleteWith(options)...)
}

func (c *controllerRuntimeClient) Get(obj *unstructured.Unstructured, options *metav1.GetOptions) (*unstructured.Unstructured, error) {
	key := client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
	result := &unstructured.Unstructured{}
	result.SetGroupVersionKind(obj.GroupVersionKind())
	err := c.client.Get(context.TODO(), key, result)
	return result, err
}

func createWith(opts *metav1.CreateOptions) []client.CreateOption {
	result := []client.CreateOption{}
	empty := &metav1.CreateOptions{}
	if len(opts.DryRun) != len(empty.DryRun) {
		result = append(result, client.DryRunAll)
	}
	if opts.FieldManager != empty.FieldManager {
		result = append(result, client.FieldOwner(opts.FieldManager))
	}
	return result
}

func updateWith(opts *metav1.UpdateOptions) []client.UpdateOption {
	result := []client.UpdateOption{}
	empty := &metav1.UpdateOptions{}
	if len(opts.DryRun) != len(empty.DryRun) {
		result = append(result, client.DryRunAll)
	}
	if opts.FieldManager != empty.FieldManager {
		result = append(result, client.FieldOwner(opts.FieldManager))
	}
	return result
}

func deleteWith(opts *metav1.DeleteOptions) []client.DeleteOption {
	result := []client.DeleteOption{}
	empty := &metav1.DeleteOptions{}
	if len(opts.DryRun) != len(empty.DryRun) {
		result = append(result, client.DryRunAll)
	}
	if opts.GracePeriodSeconds != empty.GracePeriodSeconds {
		result = append(result, client.GracePeriodSeconds(*opts.GracePeriodSeconds))
	}
	if opts.Preconditions != empty.Preconditions {
		result = append(result, client.Preconditions(*opts.Preconditions))
	}
	if opts.PropagationPolicy != empty.PropagationPolicy {
		result = append(result, client.PropagationPolicy(*opts.PropagationPolicy))
	}
	return result
}
