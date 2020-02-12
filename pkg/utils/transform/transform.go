package transform

import (
	"fmt"
	"strings"

	mf "github.com/jcrossley3/manifestival"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// InjectDefaultSA adds default service account into config-defaults configMap
func InjectDefaultSA(defaultSA string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		if strings.ToLower(u.GetKind()) != "configmap" {
			return nil
		}
		if u.GetName() != "config-defaults" {
			return nil
		}

		cm := &corev1.ConfigMap{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, cm)
		if err != nil {
			return err
		}

		cm.Data["default-service-account"] = defaultSA
		unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cm)
		if err != nil {
			return err
		}

		u.SetUnstructuredContent(unstrObj)
		return nil
	}
}

func InjectNamespaceConditional(preserveNamespace, targetNamespace string) mf.Transformer {
	tf := mf.InjectNamespace(targetNamespace)
	return func(u *unstructured.Unstructured) error {
		annotations := u.GetAnnotations()
		val, ok := annotations[preserveNamespace]
		if ok && val == "true" {
			return nil
		}

		return tf(u)
	}
}

func Kind(fromKind, toKind string) mf.Transformer {
	return func(u *unstructured.Unstructured) error {
		kind, found, err := unstructured.NestedString(u.Object, "kind")
		if err != nil || !found {
			return fmt.Errorf("cound not get resource KIND, %q", err)
		}
		if kind != fromKind {
			return nil
		}
		err = unstructured.SetNestedField(u.Object, toKind, "kind")
		if err != nil {
			return fmt.Errorf("cound change resource KIND, %q", err)
		}
		return nil
	}
}
