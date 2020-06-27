package manifestival

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testing"
	"github.com/manifestival/manifestival/patch"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Manifestival allows group application of a set of Kubernetes resources
// (typically, a set of YAML files, aka a manifest) against a Kubernetes
// apiserver.
type Manifestival interface {
	// Either updates or creates all resources in the manifest
	ApplyAll(opts ...ClientOption) error
	// Updates or creates a particular resource
	Apply(spec *unstructured.Unstructured, opts ...ClientOption) error
	// Deletes all resources in the manifest
	DeleteAll(opts ...ClientOption) error
	// Deletes a particular resource
	Delete(spec *unstructured.Unstructured, opts ...ClientOption) error
	// Returns a copy of the resource from the api server, nil if not found
	Get(spec *unstructured.Unstructured, opts ...ClientOption) (*unstructured.Unstructured, error)
	// Transforms the resources within a Manifest
	Transform(fns ...Transformer) (*Manifest, error)
}

// Manifest tracks a set of concrete resources which should be managed as a
// group using a Kubernetes client provided by `NewManifest`.
type Manifest struct {
	Resources []unstructured.Unstructured
	client    Client
	log       logr.Logger
}

var _ Manifestival = &Manifest{}

// NewManifest creates a Manifest from a comma-separated set of yaml files or
// directories (and subdirectories if the `recursive` option is set). The
// Manifest will be evaluated using the supplied `config` against a particular
// Kubernetes apiserver.
func NewManifest(pathname string, opts ...Option) (Manifest, error) {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	log := o.logger
	if log == nil {
		log = testing.NullLogger{}
	}
	log.Info("Reading manifest", "name", pathname)
	resources, err := Parse(pathname, o.recursive)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{Resources: resources, client: o.client, log: log}, nil
}

// ApplyAll updates or creates all resources in the manifest.
func (f *Manifest) ApplyAll(opts ...ClientOption) error {
	for _, spec := range f.Resources {
		if err := f.Apply(&spec, opts...); err != nil {
			return err
		}
	}
	return nil
}

// Apply updates or creates a particular resource, which does not need to be
// part of `Resources`, and will not be tracked.
func (f *Manifest) Apply(spec *unstructured.Unstructured, opts ...ClientOption) error {
	current, err := f.Get(spec, opts...)
	if err != nil {
		return err
	}
	options := NewOptions(opts...)
	if current == nil {
		f.logResource("Creating", spec)
		annotate(spec, v1.LastAppliedConfigAnnotation, patch.MakeLastAppliedConfig(spec))
		annotate(spec, "manifestival", resourceCreated)
		if err = f.client.Create(spec.DeepCopy(), options.ForCreate()); err != nil {
			return err
		}
	} else {
		patch, err := patch.NewPatch(spec, current)
		if err != nil {
			return err
		}
		if patch.IsRequired() {
			f.log.Info("Merging", "diff", patch)
			if err := patch.Merge(current); err != nil {
				return err
			}
			f.logResource("Updating", current)
			if err = f.client.Update(current, options.ForUpdate()); err != nil {
				return err
			}
		}
	}
	return nil
}

// DeleteAll removes all tracked `Resources` in the Manifest.
func (f *Manifest) DeleteAll(opts ...ClientOption) error {
	a := make([]unstructured.Unstructured, len(f.Resources))
	copy(a, f.Resources)
	// we want to delete in reverse order
	for left, right := 0, len(a)-1; left < right; left, right = left+1, right-1 {
		a[left], a[right] = a[right], a[left]
	}
	for _, spec := range a {
		if okToDelete(&spec) {
			if err := f.Delete(&spec, opts...); err != nil {
				f.log.Error(err, "Delete failed")
			}
		}
	}
	return nil
}

// Delete removes the specified objects, which do not need to be registered as
// `Resources` in the Manifest.
func (f *Manifest) Delete(spec *unstructured.Unstructured, opts ...ClientOption) error {
	current, err := f.Get(spec, opts...)
	if current == nil && err == nil {
		return nil
	}
	f.logResource("Deleting", spec)
	options := NewOptions(opts...)
	if err := f.client.Delete(spec, options.ForDelete()); err != nil {
		// ignore GC race conditions triggered by owner references
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// Get collects a full resource body (or `nil`) from a partial resource
// supplied in `spec`.
func (f *Manifest) Get(spec *unstructured.Unstructured, opts ...ClientOption) (*unstructured.Unstructured, error) {
	options := NewOptions(opts...)
	result, err := f.client.Get(spec, options.ForGet())
	if err != nil {
		result = nil
		if errors.IsNotFound(err) {
			err = nil
		}
	}
	return result, err
}

func (f *Manifest) logResource(msg string, spec *unstructured.Unstructured) {
	name := fmt.Sprintf("%s/%s", spec.GetNamespace(), spec.GetName())
	f.log.Info(msg, "name", name, "type", spec.GroupVersionKind())
}

func annotate(spec *unstructured.Unstructured, key string, value string) {
	annotations := spec.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[key] = value
	spec.SetAnnotations(annotations)
}

func okToDelete(spec *unstructured.Unstructured) bool {
	switch spec.GetKind() {
	case "Namespace":
		return spec.GetAnnotations()["manifestival"] == resourceCreated
	}
	return true
}

const (
	resourceCreated = "new"
)
