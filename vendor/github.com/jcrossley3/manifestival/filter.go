package manifestival

import (
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type FilterFn func(u *unstructured.Unstructured) bool

type Owner interface {
	v1.Object
	schema.ObjectKind
}

func (f *YamlManifest) Filter(fns ...FilterFn) Manifest {
	var results []unstructured.Unstructured
OUTER:
	for i := 0; i < len(f.resources); i++ {
		spec := f.resources[i].DeepCopy()
		for _, f := range fns {
			if !f(spec) {
				continue OUTER
			}
		}
		results = append(results, *spec)
	}
	f.resources = results
	return f
}

func ByNamespace(ns string) FilterFn {
	return func(u *unstructured.Unstructured) bool {
		if strings.ToLower(u.GetKind()) == "namespace" {
			return false
		}
		if !isClusterScoped(u.GetKind()) {
			u.SetNamespace(ns)
		}
		return true
	}
}

func ByOwner(owner Owner) FilterFn {
	return func(u *unstructured.Unstructured) bool {
		if !isClusterScoped(u.GetKind()) {
			// apparently reference counting for cluster-scoped
			// resources is broken, so trust the GC only for ns-scoped
			// dependents
			u.SetOwnerReferences([]v1.OwnerReference{*v1.NewControllerRef(owner, owner.GroupVersionKind())})
		}
		return true
	}
}

func ByOLM(u *unstructured.Unstructured) bool {
	switch strings.ToLower(u.GetKind()) {
	case "namespace", "role", "rolebinding",
		"clusterrole", "clusterrolebinding",
		"customresourcedefinition", "serviceaccount":
		return false
	}
	return true
}

func isClusterScoped(kind string) bool {
	// TODO: something more clever using !APIResource.Namespaced maybe?
	switch strings.ToLower(kind) {
	case "componentstatus",
		"namespace",
		"node",
		"persistentvolume",
		"mutatingwebhookconfiguration",
		"validatingwebhookconfiguration",
		"customresourcedefinition",
		"apiservice",
		"meshpolicy",
		"tokenreview",
		"selfsubjectaccessreview",
		"selfsubjectrulesreview",
		"subjectaccessreview",
		"certificatesigningrequest",
		"podsecuritypolicy",
		"clusterrolebinding",
		"clusterrole",
		"priorityclass",
		"storageclass",
		"volumeattachment":
		return true
	}
	return false
}
