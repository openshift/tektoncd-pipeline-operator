package manifestival

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Predicate returns true if u should be included in result
type Predicate func(u *unstructured.Unstructured) bool

// Filter returns a Manifest containing only the resources for which
// *all* Predicates return true. Any changes callers make to the
// resources passed to their Predicate[s] will only be reflected in
// the returned Manifest.
func (m Manifest) Filter(fns ...Predicate) Manifest {
	result := m
	result.resources = []unstructured.Unstructured{}
NEXT_RESOURCE:
	for _, spec := range m.Resources() {
		for _, pred := range fns {
			if pred != nil {
				if !pred(&spec) {
					continue NEXT_RESOURCE
				}
			}
		}
		result.resources = append(result.resources, spec)
	}
	return result
}

// JustCRDs returns only CustomResourceDefinitions
var JustCRDs = ByKind("CustomResourceDefinition")

// NotCRDs returns no CustomResourceDefinitions
var NotCRDs = Complement(JustCRDs)

// ByName returns resources with a specifc name
func ByName(name string) Predicate {
	return func(u *unstructured.Unstructured) bool {
		return u.GetName() == name
	}
}

// ByKind returns resources matching a particular kind
func ByKind(kind string) Predicate {
	return func(u *unstructured.Unstructured) bool {
		return u.GetKind() == kind
	}
}

// ByLabel returns resources that contain a particular label and
// value. A value of "" denotes *ANY* value
func ByLabel(label, value string) Predicate {
	return func(u *unstructured.Unstructured) bool {
		v, ok := u.GetLabels()[label]
		if value == "" {
			return ok
		}
		return v == value
	}
}

// ByGVK returns resources of a particular GroupVersionKind
func ByGVK(gvk schema.GroupVersionKind) Predicate {
	return func(u *unstructured.Unstructured) bool {
		return u.GroupVersionKind() == gvk
	}
}

// Complement returns what another Predicate wouldn't
func Complement(p Predicate) Predicate {
	return func(u *unstructured.Unstructured) bool {
		return !p(u)
	}
}
