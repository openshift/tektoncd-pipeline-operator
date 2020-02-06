package transform

import (
	"github.com/tektoncd/operator/pkg/flag"
	"testing"

	mf "github.com/jcrossley3/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformManifest_WithAnnotation(t *testing.T) {
	resourceWithAnnotation := "testdata/with-annotation.yaml"
	manifest, err := mf.NewManifest(resourceWithAnnotation, true, nil)
	if err != nil {
		t.Errorf("NewManifest() = %v, wanted no error", err)
	}

	tfs := []mf.Transformer{
		InjectNamespaceConditional(flag.AnnotationPreserveNS, "target"),
	}

	if err := manifest.Transform(tfs...); err != nil {
		t.Error("wanted no error :", err)
	}

	assertNamespace(t, manifest.Resources[0], "openshift")
}

func TestTransformManifest_WithoutAnnotation(t *testing.T) {
	resourceWithoutAnnotation := "testdata/without-annotation.yaml"
	manifest, err := mf.NewManifest(resourceWithoutAnnotation, true, nil)
	if err != nil {
		t.Errorf("NewManifest() = %v, wanted no error", err)
	}

	tfs := []mf.Transformer{
		InjectNamespaceConditional(flag.AnnotationPreserveNS, "target"),
	}

	if err := manifest.Transform(tfs...); err != nil {
		t.Error("expected no error :", err)
	}

	assertNamespace(t, manifest.Resources[0], "target")
}

func assertNamespace(t *testing.T, u unstructured.Unstructured, expected string) {
	t.Helper()
	v, _, _ := unstructured.NestedMap(u.Object, "metadata")
	ns := v["namespace"]
	if ns != expected {
		t.Errorf("Expected '%s', got '%s'", expected, ns)
	}
}
