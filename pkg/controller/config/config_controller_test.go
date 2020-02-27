package config

import (
	"context"
	"os"
	"path"
	"path/filepath"
	rt "runtime"
	"strings"
	"testing"

	mf "github.com/jcrossley3/manifestival"
	securityv1 "github.com/openshift/api/security/v1"
	fakesecurityclient "github.com/openshift/client-go/security/clientset/versioned/fake"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/flag"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestConfigControllerReplaceImages(t *testing.T) {
	t.Run("for_pipelines", func(t *testing.T) {
		var (
			configName = "cluster"
			namespace  = "openshift-pipelines"
			deployment = "tekton-pipelines-controller"
			container  = "tekton_triggers_controller"
			image      = "registry.redhat.io/osbs/controller:latest"
			arg        = "shell_image"
			argImage   = "registry.redhat.io/osbs/ubi8:latest"
		)

		// GIVEN
		os.Setenv("IMAGE_PIPELINE_"+strings.ToUpper(container), image)
		os.Setenv("IMAGE_PIPELINE_ARG_"+strings.ToUpper(arg), argImage)
		config := newConfig(configName, namespace)
		cl := feedConfigMock(config)
		secCl := feedSCCMock("privileged", t)
		pipelines, err := mfFor("pipelines", cl)
		assertNoEror(err, "failed to create manifestival for pipelines;", t)
		req := newRequest(configName, namespace)
		r := ReconcileConfig{scheme: scheme.Scheme, client: cl, pipeline: pipelines, secClient: secCl}

		// WHEN
		_, err = r.applyPipeline(req, config)

		// THEN
		assertNoEror(err, "failed to reconcile for applyPipeline;", t)
		assertContainerHasImage(deployment, container, image, r.client, t)
		assertContainerArgHasImage(deployment, arg, argImage, r.client, t)
	})

	t.Run("for_triggers_addon", func(t *testing.T) {
		var (
			configName = "cluster"
			namespace  = "openshift-pipelines"
			deployment = "tekton-triggers-controller"
			container  = "tekton_triggers_controller"
			image      = "registry.redhat.io/osbs/controller:latest"
			arg        = "el_image"
			argImage   = "registry.redhat.io/osbs/eventlistenersink:latest"
		)

		// GIVEN
		os.Setenv("IMAGE_TRIGGERS_"+strings.ToUpper(container), image)
		os.Setenv("IMAGE_TRIGGERS_ARG_"+strings.ToUpper(arg), argImage)
		config := newConfig(configName, namespace)
		cl := feedConfigMock(config)
		addons, err := mfFor("addons", cl)
		assertNoEror(err, "failed to create manifestival for triggers addons;", t)
		req := newRequest(configName, namespace)
		r := ReconcileConfig{scheme: scheme.Scheme, client: cl, addons: addons}

		// WHEN
		_, err = r.applyAddons(req, config)

		// THEN
		assertNoEror(err, "failed to reconcile for applyPipeline;", t)
		assertContainerHasImage(deployment, container, image, r.client, t)
		assertContainerArgHasImage(deployment, arg, argImage, r.client, t)
	})
}

func newConfig(name string, namespace string) *op.Config {
	return &op.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: op.ConfigSpec{
			TargetNamespace: namespace,
		},
	}
}

func newRequest(configName string, namespace string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      configName,
			Namespace: namespace,
		},
	}
}

func assertNoEror(err error, msg string, t *testing.T) {
	t.Helper()

	if err != nil {
		t.Errorf("%s expected no error, %v", msg, err)
	}
}

func assertContainerHasImage(deploy string, container string, image string, cl client.Client, t *testing.T) {
	t.Helper()

	containers := deploymentFor(deploy, cl, t)

	for _, c := range containers {
		if c.Name != container {
			continue
		}

		if c.Image != image {
			t.Fatalf("assertion failed; expected image %s but got %s", image, c.Image)
		}
	}
}

func assertContainerArgHasImage(deploy string, arg string, image string, cl client.Client, t *testing.T) {
	t.Helper()

	containers := deploymentFor(deploy, cl, t)

	for _, container := range containers {
		if container.Args == nil || len(container.Args) == 0 {
			continue
		}

		for a, argument := range container.Args {
			if argument != arg {
				continue
			}

			if container.Args[a+1] != image {
				t.Fatalf("assertion failed; expected arg image %s but got %s", image, container.Args[a+1])
			}
		}
	}
}

func deploymentFor(name string, cl client.Client, t *testing.T) []v1.Container {
	dep := &appsv1.Deployment{}
	namespacedName := types.NamespacedName{
		Name:      name,
		Namespace: "openshift-pipelines",
	}

	err := cl.Get(context.TODO(), namespacedName, dep)
	if err != nil {
		t.Fatalf("assertion failed; get deployment: (%v)", err)
	}

	containers := dep.Spec.Template.Spec.Containers

	if containers == nil && len(containers) == 0 {
		t.Fatalf("assertion failed; deployment not containers")
	}

	return containers
}

func mfFor(resource string, cl client.Client) (mf.Manifest, error) {
	_, filename, _, _ := rt.Caller(0)
	root := path.Join(path.Dir(filename), "../../..")
	pipelinePath := filepath.Join(root, flag.ResourceDir, resource)
	return mf.NewManifest(pipelinePath, flag.Recursive, cl)
}

func feedSCCMock(scc string, t *testing.T) *fakesecurityclient.Clientset {
	constraint := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{Name: scc, Annotations: map[string]string{}},
		Groups:     []string{},
	}
	securityFake := fakesecurityclient.NewSimpleClientset(constraint)
	_, err := securityFake.SecurityV1().SecurityContextConstraints().Create(constraint)
	if err != nil {
		t.Errorf("mocking failed, cant create SCC: %v", err)
	}

	return securityFake
}

func feedConfigMock(config *op.Config) client.Client {
	objs := []runtime.Object{config}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(op.SchemeGroupVersion, config)

	// Create a fake client to mock API calls.
	return fake.NewFakeClientWithScheme(s, objs...)
}
