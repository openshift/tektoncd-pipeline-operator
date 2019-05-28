package manifestival

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log = logf.Log.WithName("manifestival")
)

type Manifestival interface {
	// Either updates or creates all resources in the manifest
	ApplyAll() error
	// Updates or creates a particular resource
	Apply(*unstructured.Unstructured) error
	// Deletes all resources in the manifest
	DeleteAll() error
	// Deletes a particular resource
	Delete(spec *unstructured.Unstructured) error
	// Transforms the resources within a Manifest; returns itself
	Transform(fns ...Transformer) *Manifest
	// Returns a deep copy of the matching resource read from the file
	Find(apiVersion string, kind string, name string) *unstructured.Unstructured
	// Returns the resource fetched from the api server, nil if not found
	Get(spec *unstructured.Unstructured) (*unstructured.Unstructured, error)
}

type Manifest struct {
	Resources []unstructured.Unstructured
	client    client.Client
}

var _ Manifestival = &Manifest{}

func NewManifest(pathname string, recursive bool, client client.Client) (Manifest, error) {
	log.Info("Reading file", "name", pathname)
	resources, err := Parse(pathname, recursive)
	if err != nil {
		return Manifest{}, err
	}
	return Manifest{Resources: resources, client: client}, nil
}

func (f *Manifest) ApplyAll() error {
	for _, spec := range f.Resources {
		if err := f.Apply(&spec); err != nil {
			return err
		}
	}
	return nil
}

func (f *Manifest) Apply(spec *unstructured.Unstructured) error {
	current, err := f.Get(spec)
	if err != nil {
		return err
	}
	if current == nil {
		logResource("Creating", spec)
		if err = f.client.Create(context.TODO(), spec); err != nil {
			return err
		}
	} else {
		// Update existing one
		if UpdateChanged(spec.UnstructuredContent(), current.UnstructuredContent()) {
			logResource("Updating", spec)
			if err = f.client.Update(context.TODO(), current); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *Manifest) DeleteAll() error {
	a := make([]unstructured.Unstructured, len(f.Resources))
	copy(a, f.Resources)
	// we want to delete in reverse order
	for left, right := 0, len(a)-1; left < right; left, right = left+1, right-1 {
		a[left], a[right] = a[right], a[left]
	}
	for _, spec := range a {
		if err := f.Delete(&spec); err != nil {
			return err
		}
	}
	return nil
}

func (f *Manifest) Delete(spec *unstructured.Unstructured) error {
	current, err := f.Get(spec)
	if current == nil && err == nil {
		return nil
	}
	logResource("Deleting", spec)
	if err := f.client.Delete(context.TODO(), spec); err != nil {
		// ignore GC race conditions triggered by owner references
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (f *Manifest) Get(spec *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	key := client.ObjectKey{Namespace: spec.GetNamespace(), Name: spec.GetName()}
	result := &unstructured.Unstructured{}
	result.SetGroupVersionKind(spec.GroupVersionKind())
	err := f.client.Get(context.TODO(), key, result)
	if err != nil {
		result = nil
		if errors.IsNotFound(err) {
			err = nil
		}
	}
	return result, err
}

func (f *Manifest) Find(apiVersion string, kind string, name string) *unstructured.Unstructured {
	for _, spec := range f.Resources {
		if spec.GetAPIVersion() == apiVersion &&
			spec.GetKind() == kind &&
			spec.GetName() == name {
			return spec.DeepCopy()
		}
	}
	return nil
}

// We need to preserve the top-level target keys, specifically
// 'metadata.resourceVersion', 'spec.clusterIP', and any existing
// entries in a ConfigMap's 'data' field. So we only overwrite fields
// set in our src resource.
func UpdateChanged(src, tgt map[string]interface{}) bool {
	changed := false
	for k, v := range src {
		if v, ok := v.(map[string]interface{}); ok {
			if tgt[k] == nil {
				tgt[k], changed = v, true
			} else if UpdateChanged(v, tgt[k].(map[string]interface{})) {
				// This could be an issue if a field in a nested src
				// map doesn't overwrite its corresponding tgt
				changed = true
			}
			continue
		}
		if !equality.Semantic.DeepEqual(v, tgt[k]) {
			tgt[k], changed = v, true
		}
	}
	return changed
}

func logResource(msg string, spec *unstructured.Unstructured) {
	name := fmt.Sprintf("%s/%s", spec.GetNamespace(), spec.GetName())
	log.Info(msg, "name", name, "type", spec.GroupVersionKind())
}
