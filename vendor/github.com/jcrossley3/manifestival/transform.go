package manifestival

import (
	"os"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Transform one into another; return nil to reject/delete
type Transformer func(u *unstructured.Unstructured) *unstructured.Unstructured

type Owner interface {
	v1.Object
	schema.ObjectKind
}

func (f *Manifest) Transform(fns ...Transformer) *Manifest {
	var results []unstructured.Unstructured
OUTER:
	for i := 0; i < len(f.Resources); i++ {
		spec := f.Resources[i].DeepCopy()
		for _, f := range fns {
			spec = f(spec)
			if spec == nil {
				continue OUTER
			}
		}
		results = append(results, *spec)
	}
	f.Resources = results
	return f
}

// We assume all resources in the manifest live in the same namespace
func InjectNamespace(ns string) Transformer {
	namespace := resolveEnv(ns)
	return func(u *unstructured.Unstructured) *unstructured.Unstructured {
		switch strings.ToLower(u.GetKind()) {
		case "namespace":
			return nil
		case "clusterrolebinding":
			subjects, _, _ := unstructured.NestedFieldNoCopy(u.Object, "subjects")
			for _, subject := range subjects.([]interface{}) {
				m := subject.(map[string]interface{})
				if _, ok := m["namespace"]; ok {
					m["namespace"] = namespace
				}
			}
		}
		if !isClusterScoped(u.GetKind()) {
			u.SetNamespace(namespace)
		}
		return u
	}
}

func InjectOwner(owner Owner) Transformer {
	return func(u *unstructured.Unstructured) *unstructured.Unstructured {
		if !isClusterScoped(u.GetKind()) {
			// apparently reference counting for cluster-scoped
			// resources is broken, so trust the GC only for ns-scoped
			// dependents
			u.SetOwnerReferences([]v1.OwnerReference{*v1.NewControllerRef(owner, owner.GroupVersionKind())})
		}
		return u
	}
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

func resolveEnv(x string) string {
	if len(x) > 1 && x[:1] == "$" {
		return os.Getenv(x[1:])
	}
	return x
}
