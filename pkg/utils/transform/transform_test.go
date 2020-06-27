package transform

import (
	"path"
	"reflect"
	"testing"

	mf "github.com/manifestival/manifestival"
	"github.com/tektoncd/operator/pkg/flag"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestTransformManifest_WithAnnotation(t *testing.T) {
	resourceWithAnnotation := "testdata/with-annotation.yaml"
	manifest, err := mf.NewManifest(resourceWithAnnotation, mf.UseRecursive(true))
	assertNoEror(t, err)
	tf := InjectNamespaceConditional(flag.AnnotationPreserveNS, "target")
	newManifest, err := manifest.Transform(tf)
	assertNoEror(t, err)
	assertNamespace(t, newManifest.Resources[0], "openshift")
}

func TestTransformManifest_WithoutAnnotation(t *testing.T) {
	resourceWithoutAnnotation := "testdata/without-annotation.yaml"
	manifest, err := mf.NewManifest(resourceWithoutAnnotation, mf.UseRecursive(true))
	assertNoEror(t, err)
	tf := InjectNamespaceConditional(flag.AnnotationPreserveNS, "target")
	newManifest, err := manifest.Transform(tf)
	assertNoEror(t, err)
	assertNamespace(t, newManifest.Resources[0], "target")
}

func TestReplaceKind(t *testing.T) {
	fromKind := "Task"
	fromKindMismatch := "Pod"
	toKind := "ClusterTask"
	testData := path.Join("testdata", "test-replace-kind.yaml")

	t.Run("should replace Kind when resource kind == fromKind", func(t *testing.T) {
		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		replaceKind := ReplaceKind(fromKind, toKind)
		newManifest, err := manifest.Transform(replaceKind)
		assertNoEror(t, err)
		assertKind(t, newManifest.Resources[0], toKind)
	})
	t.Run("should not replace Kind when resource kind != fromKind", func(t *testing.T) {
		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		replaceKind := ReplaceKind(fromKindMismatch, toKind)
		newManifest, err := manifest.Transform(replaceKind)
		assertNoEror(t, err)
		assertKind(t, newManifest.Resources[0], fromKind)
	})
}

func assertKind(t *testing.T, u unstructured.Unstructured, kind string) {
	t.Helper()
	if k := u.GetKind(); k != kind {
		t.Errorf("expected kind %s, got kind %s", kind, k)
	}
}

func TestInjectLabel(t *testing.T) {
	key := flag.LabelProviderType
	value := flag.ProviderTypeCommunity

	t.Run("should add label to a resource", func(t *testing.T) {
		testData := path.Join("testdata", "test-inject-label.yaml")

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		injectLabel := InjectLabel(key, value, Overwrite)
		newManifest, err := manifest.Transform(injectLabel)
		assertNoEror(t, err)
		assertLabel(t, newManifest.Resources[0], key, value)
	})
	t.Run("should add label if kind(s) is specified and does not match resource kind", func(t *testing.T) {
		testData := path.Join("testdata", "test-inject-label.yaml")

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		injectLabel := InjectLabel(key, value, Overwrite, "Service")
		newManifest, err := manifest.Transform(injectLabel)
		assertNoEror(t, err)
		assertNoLabel(t, newManifest.Resources[0], key, value)
	})

	t.Run("should retain original label with overwritePolicy 'Retain'", func(t *testing.T) {
		existingValue := flag.ProviderTypeRedHat
		testData := path.Join("testdata", "test-inject-label-overwrite.yaml")

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		injectLabel := InjectLabel(key, value, Retain)
		newManifest, err := manifest.Transform(injectLabel)
		assertNoEror(t, err)
		assertLabel(t, newManifest.Resources[0], key, existingValue)
	})
	t.Run("should overwrite original label with overwritePolicy 'Overwrite'", func(t *testing.T) {
		testData := path.Join("testdata", "test-inject-label-overwrite.yaml")

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		injectLabel := InjectLabel(key, value, Overwrite)
		newManifest, err := manifest.Transform(injectLabel)
		assertNoEror(t, err)
		assertLabel(t, newManifest.Resources[0], key, value)
	})
	t.Run("should add labels only to specified resources", func(t *testing.T) {
		testData := path.Join("testdata", "test-inject-label-kind-set.yaml")
		kinds := []string{
			"Pod",
			"Service",
		}

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		injectLabel := InjectLabel(key, value, Overwrite, kinds...)
		newManifest, err := manifest.Transform(injectLabel)
		assertNoEror(t, err)
		assertOnResourceList(t, newManifest.Resources, key, value, kinds...)
	})
}

func TestReplaceImages(t *testing.T) {
	t.Run("ignore_non_deployment", func(t *testing.T) {
		testData := path.Join("testdata", "test-replace-kind.yaml")
		expected, _ := mf.NewManifest(testData, mf.UseRecursive(true))

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		newManifest, err := manifest.Transform(DeploymentImages(map[string]string{}))
		assertNoEror(t, err)
		assertEqual(t, newManifest.Resources, expected.Resources)
	})

	t.Run("of_containers_by_name", func(t *testing.T) {
		image := "foo.bar/image/controller"
		images := map[string]string{
			"controller_deployment": image,
		}
		testData := path.Join("testdata", "test-replace-image.yaml")

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		newManifest, err := manifest.Transform(DeploymentImages(images))
		assertNoEror(t, err)
		assertDeployContainersHasImage(t, newManifest.Resources, "controller", image)
		assertDeployContainersHasImage(t, newManifest.Resources, "sidecar", "busybox")
	})

	t.Run("of_containers_args_by_space", func(t *testing.T) {
		arg := ArgPrefix + "__bash_image"
		image := "foo.bar/image/bash"
		images := map[string]string{
			arg: image,
		}
		testData := path.Join("testdata", "test-replace-image.yaml")

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		newManifest, err := manifest.Transform(DeploymentImages(images))
		assertNoEror(t, err)
		assertDeployContainerArgsHasImage(t, newManifest.Resources, "-bash", image)
		assertDeployContainerArgsHasImage(t, newManifest.Resources, "-git", "git")
	})

	t.Run("of_container_args_has_equal", func(t *testing.T) {
		arg := ArgPrefix + "__nop"
		image := "foo.bar/image/nop"
		images := map[string]string{
			arg: image,
		}
		testData := path.Join("testdata", "test-replace-image.yaml")

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		newManifest, err := manifest.Transform(DeploymentImages(images))
		assertNoEror(t, err)
		assertDeployContainerArgsHasImage(t, newManifest.Resources, "-nop", image)
		assertDeployContainerArgsHasImage(t, newManifest.Resources, "-git", "git")
	})

	t.Run("of_task_addons_step_image", func(t *testing.T) {
		stepName := "push_image"
		image := "foo.bar/image/buildah"
		images := map[string]string{
			stepName: image,
		}
		testData := path.Join("testdata", "test-replace-addon-image.yaml")

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		newManifest, err := manifest.Transform(TaskImages(images))
		assertNoEror(t, err)
		assertTaskImage(t, newManifest.Resources, "push", image)
		assertTaskImage(t, newManifest.Resources, "build", "$(inputs.params.BUILDER_IMAGE)")
	})

	t.Run("of_task_addons_param_image", func(t *testing.T) {
		paramName := ParamPrefix + "builder_image"
		image := "foo.bar/image/buildah"
		images := map[string]string{
			paramName: image,
		}
		testData := path.Join("testdata", "test-replace-addon-image.yaml")

		manifest, err := mf.NewManifest(testData, mf.UseRecursive(true))
		assertNoEror(t, err)
		newManifest, err := manifest.Transform(TaskImages(images))
		assertNoEror(t, err)
		assertParamHasImage(t, newManifest.Resources, "BUILDER_IMAGE", image)
		assertTaskImage(t, newManifest.Resources, "push", "buildah")
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

func assertLabel(t *testing.T, u unstructured.Unstructured, key, value string) {
	t.Helper()
	labels, found, err := unstructured.NestedStringMap(u.Object, "metadata", "labels")
	assertNoEror(t, err)
	got, ok := labels[key]
	if !found || !ok || got != value {
		t.Errorf("expected %s, got %s", value, got)
	}
}

func assertNoLabel(t *testing.T, u unstructured.Unstructured, key, value string) {
	t.Helper()
	labels, found, err := unstructured.NestedStringMap(u.Object, "metadata", "labels")
	assertNoEror(t, err)
	got, ok := labels[key]
	if found && ok && got == value {
		t.Errorf("not expected %s, got %s", value, got)
	}
}

func assertOnResourceList(t *testing.T, items []unstructured.Unstructured, key, value string, kinds ...string) {
	t.Helper()
	for _, item := range items {
		k := item.GetKind()
		if ItemInSlice(k, kinds) {
			assertLabel(t, item, key, value)
		} else {
			assertNoLabel(t, item, key, value)
		}
	}
}

func assertNoEror(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("assertion failed; expected no error %v", err)
	}
}

func assertEqual(t *testing.T, result []unstructured.Unstructured, expected []unstructured.Unstructured) {
	t.Helper()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("assertion failed; not equal: expected %v, got %v", expected, result)
	}
}

func assertDeployContainersHasImage(t *testing.T, resources []unstructured.Unstructured, name string, image string) {
	t.Helper()

	for _, resource := range resources {
		deployment := deploymentFor(t, resource)
		containers := deployment.Spec.Template.Spec.Containers

		for _, container := range containers {
			if container.Name != name {
				continue
			}

			if container.Image != image {
				t.Errorf("assertion failed; unexpected image: expected %s and got %s", image, container.Image)
			}
		}
	}
}

func assertDeployContainerArgsHasImage(t *testing.T, resources []unstructured.Unstructured, arg string, image string) {
	t.Helper()

	for _, resource := range resources {
		deployment := deploymentFor(t, resource)
		containers := deployment.Spec.Template.Spec.Containers

		for _, container := range containers {
			if len(container.Args) == 0 {
				continue
			}

			for a, argument := range container.Args {
				if argument == arg && container.Args[a+1] != image {
					t.Errorf("not equal: expected %v, got %v", image, container.Args[a+1])
				}
			}
		}
	}
}

func deploymentFor(t *testing.T, unstr unstructured.Unstructured) *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstr.Object, deployment)
	if err != nil {
		t.Errorf("failed to load deployment yaml")
	}
	return deployment
}

func assertParamHasImage(t *testing.T, resources []unstructured.Unstructured, name string, image string) {
	t.Helper()

	for _, r := range resources {
		params, found, err := unstructured.NestedSlice(r.Object, "spec", "params")
		if err != nil {
			t.Errorf("assertion failed; %v", err)
		}
		if !found {
			continue
		}

		for _, p := range params {
			param := p.(map[string]interface{})
			n, ok := param["name"].(string)
			if !ok {
				t.Errorf("assertion failed; step name not found")
				continue
			}
			if n != name {
				continue
			}

			i, ok := param["default"].(string)
			if !ok {
				t.Errorf("assertion failed; default image not found")
				continue
			}
			if i != image {
				t.Errorf("assertion failed; unexpected image: expected %s, got %s", image, i)
			}
		}
	}
}

func assertTaskImage(t *testing.T, resources []unstructured.Unstructured, name string, image string) {
	t.Helper()

	for _, r := range resources {
		steps, found, err := unstructured.NestedSlice(r.Object, "spec", "steps")
		if err != nil {
			t.Errorf("assertion failed; %v", err)
		}
		if !found {
			continue
		}

		for _, s := range steps {
			step := s.(map[string]interface{})
			n, ok := step["name"].(string)
			if !ok {
				t.Errorf("assertion failed; step name not found")
				continue
			}
			if n != name {
				continue
			}

			i, ok := step["image"].(string)
			if !ok {
				t.Errorf("assertion failed; image not found")
				continue
			}
			if i != image {
				t.Errorf("assertion failed; unexpected image: expected %s, got %s", image, i)
			}
		}
	}
}

func TestTransformManifest_InjectNamespaceRoleBindingSubjects(t *testing.T) {
	resourceWithAnnotation := "testdata/inject-ns-rolebinding.yaml"
	manifest, err := mf.NewManifest(resourceWithAnnotation, mf.UseRecursive(true))
	assertNoEror(t, err)
	tf := InjectNamespaceRoleBindingSubjects("target")
	newManifest, err := manifest.Transform(tf)
	assertNoEror(t, err)
	assertRBSubjectNamespace(t, newManifest.Resources[0], "target")
}

func TestTransformManifest_InjectNamespaceRoleBindingSubjects_NoNs(t *testing.T) {
	resourceWithAnnotation := "testdata/inject-ns-rolebinding-no-ns.yaml"
	manifest, err := mf.NewManifest(resourceWithAnnotation, mf.UseRecursive(true))
	assertNoEror(t, err)
	tf := InjectNamespaceRoleBindingSubjects("target")
	newManifest, err := manifest.Transform(tf)
	assertNoEror(t, err)
	assertRBSubjectNoNamespace(t, newManifest.Resources[0])
}

func TestTransformManifest_InjectNamespaceCRDWebhookClientConf(t *testing.T) {
	resourceWithAnnotation := "testdata/inject-ns-crd-webhookclientconf.yaml"
	manifest, err := mf.NewManifest(resourceWithAnnotation, mf.UseRecursive(true))
	assertNoEror(t, err)
	tf := InjectNamespaceCRDWebhookClientConfig("target")
	newManifest, err := manifest.Transform(tf)
	assertNoEror(t, err)
	assertCRDWebhookClientConfNS(t, newManifest.Resources[0], "target")
}

func TestTransformManifest_InjectNamespaceCRDWebhookClientConf_NoNs(t *testing.T) {
	resourceWithAnnotation := "testdata/inject-ns-crd-webhookclientconf-no-ns.yaml"
	manifest, err := mf.NewManifest(resourceWithAnnotation, mf.UseRecursive(true))
	assertNoEror(t, err)
	tf := InjectNamespaceCRDWebhookClientConfig("target")
	newManifest, err := manifest.Transform(tf)
	assertNoEror(t, err)
	assertCRDWebhookClientConfNoNS(t, newManifest.Resources[0])
}

func assertRBSubjectNamespace(t *testing.T, u unstructured.Unstructured, expected string) {
	t.Helper()
	subjects, found, _ := unstructured.NestedFieldNoCopy(u.Object, "subjects")
	if found {
		for _, subject := range subjects.([]interface{}) {
			m := subject.(map[string]interface{})
			if _, ok := m["namespace"]; !ok {
				t.Errorf("Namespace not found")
			}
			ns := m["namespace"]
			if ns != expected {
				t.Errorf("Expected '%s', got '%s'", expected, ns)
			}
		}
	}
}

func assertRBSubjectNoNamespace(t *testing.T, u unstructured.Unstructured) {
	t.Helper()
	subjects, found, _ := unstructured.NestedFieldNoCopy(u.Object, "subjects")
	if found {
		for _, subject := range subjects.([]interface{}) {
			m := subject.(map[string]interface{})
			if _, ok := m["namespace"]; ok {
				t.Errorf("Expected no namesapce")
			}
		}
	}
}

func assertCRDWebhookClientConfNS(t *testing.T, u unstructured.Unstructured, expected string) {
	t.Helper()
	service, found, _ := unstructured.NestedFieldNoCopy(u.Object, "spec", "conversion", "webhookClientConfig", "service")
	if found {
		m := service.(map[string]interface{})
		if _, ok := m["namespace"]; !ok {
			t.Errorf("Namespace not found")
		}
		ns := m["namespace"]
		if ns != expected {
			t.Errorf("Expected '%s', got '%s'", expected, ns)
		}
	}
}

func assertCRDWebhookClientConfNoNS(t *testing.T, u unstructured.Unstructured) {
	t.Helper()
	service, found, _ := unstructured.NestedFieldNoCopy(u.Object, "spec", "conversion", "webhookClientConfig", "service")
	if found {
		m := service.(map[string]interface{})
		if _, ok := m["namespace"]; ok {
			t.Errorf("Expected no namesapce")
		}
	}
}
