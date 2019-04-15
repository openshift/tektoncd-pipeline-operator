package manifestival

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log = logf.Log.WithName("manifestival")
)

type Manifest interface {
	ApplyAll() error
	Apply(*unstructured.Unstructured) error
	DeleteAll() error
	Delete(spec *unstructured.Unstructured) error
	Filter(fns ...FilterFn) Manifest
	Find(apiVersion string, kind string, name string) *unstructured.Unstructured
	DeepCopyResources() []unstructured.Unstructured
	ResourceNames() []string
	ResourceInterface(spec *unstructured.Unstructured) (dynamic.ResourceInterface, error)
}

type YamlManifest struct {
	dynamicClient dynamic.Interface
	resources     []unstructured.Unstructured
}

var _ Manifest = &YamlManifest{}

func NewYamlManifest(pathname string, recursive bool, config *rest.Config) Manifest {
	client, _ := dynamic.NewForConfig(config)
	log.Info("Reading YAML file", "name", pathname)
	return &YamlManifest{resources: Parse(pathname, recursive), dynamicClient: client}
}

func (f *YamlManifest) ApplyAll() error {
	for _, spec := range f.resources {
		if err := f.Apply(&spec); err != nil {
			return err
		}
	}
	return nil
}

func (f *YamlManifest) Apply(spec *unstructured.Unstructured) error {
	resource, err := f.ResourceInterface(spec)
	if err != nil {
		return err
	}
	current, err := resource.Get(spec.GetName(), v1.GetOptions{})
	if err != nil {
		// Create new one
		if !errors.IsNotFound(err) {
			return err
		}
		log.Info("Creating", "type", spec.GroupVersionKind(), "name", spec.GetName())
		if _, err = resource.Create(spec, v1.CreateOptions{}); err != nil {
			return err
		}
	} else {
		// Update existing one
		log.Info("Updating", "type", spec.GroupVersionKind(), "name", spec.GetName())
		// We need to preserve the current content, specifically
		// 'metadata.resourceVersion' and 'spec.clusterIP', so we
		// only overwrite fields set in our resource
		content := current.UnstructuredContent()
		for k, v := range spec.UnstructuredContent() {
			if k == "metadata" || k == "spec" {
				m := v.(map[string]interface{})
				for kn, vn := range m {
					unstructured.SetNestedField(content, vn, k, kn)
				}
			} else {
				content[k] = v
			}
		}
		current.SetUnstructuredContent(content)
		if _, err = resource.Update(current, v1.UpdateOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (f *YamlManifest) DeleteAll() error {
	a := make([]unstructured.Unstructured, len(f.resources))
	copy(a, f.resources)
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

func (f *YamlManifest) Delete(spec *unstructured.Unstructured) error {
	resource, err := f.ResourceInterface(spec)
	if err != nil {
		return err
	}
	if _, err = resource.Get(spec.GetName(), v1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
	}
	log.Info("Deleting", "type", spec.GroupVersionKind(), "name", spec.GetName())
	if err = resource.Delete(spec.GetName(), &v1.DeleteOptions{}); err != nil {
		// ignore GC race conditions triggered by owner references
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (f *YamlManifest) Find(apiVersion string, kind string, name string) *unstructured.Unstructured {
	for _, spec := range f.resources {
		if spec.GetAPIVersion() == apiVersion &&
			spec.GetKind() == kind &&
			spec.GetName() == name {
			return spec.DeepCopy()
		}
	}
	return nil
}

func (f *YamlManifest) DeepCopyResources() []unstructured.Unstructured {
	result := make([]unstructured.Unstructured, len(f.resources))
	for i, spec := range f.resources {
		result[i] = *spec.DeepCopy()
	}
	return result
}

func (f *YamlManifest) ResourceNames() []string {
	var names []string
	for _, spec := range f.resources {
		names = append(names, fmt.Sprintf("%s/%s (%s)", spec.GetNamespace(), spec.GetName(), spec.GroupVersionKind()))
	}
	return names
}

func (f *YamlManifest) ResourceInterface(spec *unstructured.Unstructured) (dynamic.ResourceInterface, error) {
	groupVersion, err := schema.ParseGroupVersion(spec.GetAPIVersion())
	if err != nil {
		return nil, err
	}
	groupVersionResource := groupVersion.WithResource(pluralize(spec.GetKind()))
	return f.dynamicClient.Resource(groupVersionResource).Namespace(spec.GetNamespace()), nil
}

func pluralize(kind string) string {
	ret := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(ret, "s"):
		return fmt.Sprintf("%ses", ret)
	case strings.HasSuffix(ret, "policy"):
		return fmt.Sprintf("%sies", ret[:len(ret)-1])
	default:
		return fmt.Sprintf("%ss", ret)
	}
}
