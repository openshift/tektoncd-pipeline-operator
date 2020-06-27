package manifestival

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Client interface {
	Create(obj *unstructured.Unstructured, options *metav1.CreateOptions) error
	Update(obj *unstructured.Unstructured, options *metav1.UpdateOptions) error
	Delete(obj *unstructured.Unstructured, options *metav1.DeleteOptions) error
	Get(obj *unstructured.Unstructured, options *metav1.GetOptions) (*unstructured.Unstructured, error)
}

// Functional options pattern
type ClientOption func(*ClientOptions)

type ClientOptions struct {
	DryRun             []string
	FieldManager       string
	GracePeriodSeconds *int64
	Preconditions      *metav1.Preconditions
	PropagationPolicy  *metav1.DeletionPropagation
	ResourceVersion    string
}

func NewOptions(opts ...ClientOption) *ClientOptions {
	result := &ClientOptions{}
	for _, opt := range opts {
		opt(result)
	}
	return result
}

func (o *ClientOptions) ForCreate() *metav1.CreateOptions {
	return &metav1.CreateOptions{
		DryRun:       o.DryRun,
		FieldManager: o.FieldManager,
	}
}

func (o *ClientOptions) ForUpdate() *metav1.UpdateOptions {
	return &metav1.UpdateOptions{
		DryRun:       o.DryRun,
		FieldManager: o.FieldManager,
	}
}

func (o *ClientOptions) ForDelete() *metav1.DeleteOptions {
	return &metav1.DeleteOptions{
		DryRun:             o.DryRun,
		GracePeriodSeconds: o.GracePeriodSeconds,
		Preconditions:      o.Preconditions,
		PropagationPolicy:  o.PropagationPolicy,
	}
}

func (o *ClientOptions) ForGet() *metav1.GetOptions {
	return &metav1.GetOptions{
		ResourceVersion: o.ResourceVersion,
	}
}

func DryRun(v []string) ClientOption {
	return func(o *ClientOptions) {
		o.DryRun = v
	}
}
func FieldManager(v string) ClientOption {
	return func(o *ClientOptions) {
		o.FieldManager = v
	}
}
func GracePeriodSeconds(v *int64) ClientOption {
	return func(o *ClientOptions) {
		o.GracePeriodSeconds = v
	}
}
func Preconditions(v *metav1.Preconditions) ClientOption {
	return func(o *ClientOptions) {
		o.Preconditions = v
	}
}
func PropagationPolicy(v *metav1.DeletionPropagation) ClientOption {
	return func(o *ClientOptions) {
		o.PropagationPolicy = v
	}
}
func ResourceVersion(v string) ClientOption {
	return func(o *ClientOptions) {
		o.ResourceVersion = v
	}
}
