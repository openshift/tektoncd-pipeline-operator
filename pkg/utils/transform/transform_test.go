package transform_test

import (
	"path"
	"testing"

	"github.com/tektoncd/operator/pkg/flag"
	"github.com/tektoncd/operator/pkg/utils/transform"

	mf "github.com/jcrossley3/manifestival"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestTransformManifest_WithAnnotation(t *testing.T) {
	resourceWithAnnotation := "testdata/with-annotation.yaml"
	manifest, err := mf.NewManifest(resourceWithAnnotation, true, nil)
	assetNoEror(t, err)
	tf := transform.InjectNamespaceConditional(flag.AnnotationPreserveNS, "target")
	err = manifest.Transform(tf)
	assetNoEror(t, err)
	assertNamespace(t, manifest.Resources[0], "openshift")
}

func TestTransformManifest_WithoutAnnotation(t *testing.T) {
	resourceWithoutAnnotation := "testdata/without-annotation.yaml"
	manifest, err := mf.NewManifest(resourceWithoutAnnotation, true, nil)
	assetNoEror(t, err)
	tf := transform.InjectNamespaceConditional(flag.AnnotationPreserveNS, "target")
	err = manifest.Transform(tf)
	assetNoEror(t, err)
	assertNamespace(t, manifest.Resources[0], "target")
}

func TestReplaceKind(t *testing.T) {
	fromKind := "Task"
	fromKindMismatch := "Pod"
	toKind := "ClusterTask"
	testData := path.Join("testdata", "test-replace-kind.yaml")

	t.Run("should replace Kind when resource kind == fromKind", func(t *testing.T) {
		manifest, err := mf.NewManifest(testData, true, nil)
		assetNoEror(t, err)
		replaceKind := transform.ReplaceKind(fromKind, toKind)
		err = manifest.Transform(replaceKind)
		assetNoEror(t, err)
		assertKind(t, manifest.Resources[0], toKind)
	})
	t.Run("should not replace Kind when resource kind != fromKind", func(t *testing.T) {
		manifest, err := mf.NewManifest(testData, true, nil)
		assetNoEror(t, err)
		replaceKind := transform.ReplaceKind(fromKindMismatch, toKind)
		err = manifest.Transform(replaceKind)
		assetNoEror(t, err)
		assertKind(t, manifest.Resources[0], fromKind)
	})
}

func TestInjectLabel(t *testing.T) {
	key := flag.LabelProviderType
	value := flag.ProviderTypeCommunity

	t.Run("should add label to a resource", func(t *testing.T) {
		testData := path.Join("testdata", "test-inject-label.yaml")

		manifest, err := mf.NewManifest(testData, true, nil)
		assetNoEror(t, err)
		injectLabel := transform.InjectLabel(key, value, transform.Overwrite)
		err = manifest.Transform(injectLabel)
		assetNoEror(t, err)
		assertLabel(t, manifest.Resources[0], key, value)
	})
	t.Run("should add label if kind(s) is specified and does not match resource kind", func(t *testing.T) {
		testData := path.Join("testdata", "test-inject-label.yaml")

		manifest, err := mf.NewManifest(testData, true, nil)
		assetNoEror(t, err)
		injectLabel := transform.InjectLabel(key, value, transform.Overwrite, "Service")
		err = manifest.Transform(injectLabel)
		assetNoEror(t, err)
		assertNoLabel(t, manifest.Resources[0], key, value)
	})

	t.Run("should retain original label with overwritePolicy 'Retain'", func(t *testing.T) {
		existingValue := flag.ProviderTypeRedHat
		testData := path.Join("testdata", "test-inject-label-overwrite.yaml")

		manifest, err := mf.NewManifest(testData, true, nil)
		assetNoEror(t, err)
		injectLabel := transform.InjectLabel(key, value, transform.Retain)
		err = manifest.Transform(injectLabel)
		assetNoEror(t, err)
		assertLabel(t, manifest.Resources[0], key, existingValue)
	})
	t.Run("should overwrite original label with overwritePolicy 'Overwrite'", func(t *testing.T) {
		testData := path.Join("testdata", "test-inject-label-overwrite.yaml")

		manifest, err := mf.NewManifest(testData, true, nil)
		assetNoEror(t, err)
		injectLabel := transform.InjectLabel(key, value, transform.Overwrite)
		err = manifest.Transform(injectLabel)
		assetNoEror(t, err)
		assertLabel(t, manifest.Resources[0], key, value)
	})
	t.Run("should add labels only to specified resources", func(t *testing.T) {
		testData := path.Join("testdata", "test-inject-label-kind-set.yaml")
		kinds := []string{
			"Pod",
			"Service",
		}

		manifest, err := mf.NewManifest(testData, true, nil)
		assetNoEror(t, err)
		injectLabel := transform.InjectLabel(key, value, transform.Overwrite, kinds...)
		err = manifest.Transform(injectLabel)
		assetNoEror(t, err)
		assertOnResourceList(t, manifest.Resources, key, value, kinds...)
	})
}

func assertNamespace(t *testing.T, u unstructured.Unstructured, expected string) {
	t.Helper()
	v, _, _ := unstructured.NestedMap(u.Object, "metadata")
	ns := v["namespace"]
	if ns != expected {
		t.Errorf("Expected '%s', got '%s'", expected, ns)
	}
}

func assetNoEror(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("expected no error, %v", err)
	}
}

func assertKind(t *testing.T, u unstructured.Unstructured, kind string) {
	t.Helper()
	if k := u.GetKind(); k != kind {
		t.Errorf("expected kind %s, got kind %s", kind, k)
	}
}

func assertLabel(t *testing.T, u unstructured.Unstructured, key, value string) {
	t.Helper()
	labels, found, err := unstructured.NestedStringMap(u.Object, "metadata", "labels")
	assetNoEror(t, err)
	got, ok := labels[key]
	if !found || !ok || got != value {
		t.Errorf("expected %s, got %s", value, got)
	}
}

func assertNoLabel(t *testing.T, u unstructured.Unstructured, key, value string) {
	t.Helper()
	labels, found, err := unstructured.NestedStringMap(u.Object, "metadata", "labels")
	assetNoEror(t, err)
	got, ok := labels[key]
	if found && ok && got == value {
		t.Errorf("not expected %s, got %s", value, got)
	}
}

func assertOnResourceList(t *testing.T, items []unstructured.Unstructured, key, value string, kinds ...string) {
	t.Helper()
	for _, item := range items {
		k := item.GetKind()
		if transform.ItemInSlice(k, kinds) {
			assertLabel(t, item, key, value)
		} else {
			assertNoLabel(t, item, key, value)
		}
	}
}
