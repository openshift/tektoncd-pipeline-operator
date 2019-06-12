package testsuites

import (
	"context"
	"testing"
	"time"

	"github.com/openshift/tektoncd-pipeline-operator/pkg/apis/tekton/v1alpha1"
	"github.com/openshift/tektoncd-pipeline-operator/test/config"
	"github.com/openshift/tektoncd-pipeline-operator/test/helpers"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateInstallCR creates an instance of install.tekton.dev
// and checks whether openshift pipelines deployment are created
func CreateInstallCR(t *testing.T) {

	t.Run("watched-namespace", createCRinWatchednamespace)
	t.Run("watched-namespace", createCRWithIngressinWatchednamespace)
}

func createCRinWatchednamespace(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	namespace, err := ctx.GetNamespace()
	helpers.AssertNoError(t, err)

	cr := &v1alpha1.Install{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Install",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipelines-install",
			Namespace: namespace,
		},
	}

	cleanup := &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       5 * time.Second,
		RetryInterval: 1 * time.Second,
	}

	f := framework.Global
	err = f.Client.Create(context.TODO(), cr, cleanup)
	helpers.AssertNoError(t, err)

	err = e2eutil.WaitForDeployment(
		t,
		f.KubeClient,
		namespace,
		"tekton-pipelines-controller",
		1,
		config.APIRetry,
		config.APITimeout,
	)
	helpers.AssertNoError(t, err)

	err = e2eutil.WaitForDeployment(
		t,
		f.KubeClient,
		namespace,
		"tekton-pipelines-webhook",
		1,
		config.APIRetry,
		config.APITimeout,
	)
	helpers.AssertNoError(t, err)


	_, err = f.KubeClient.ExtensionsV1beta1().Ingresses(namespace).Get("tekton-dashboard", metav1.GetOptions{})
	if err == nil {
		t.Error("Ingress should not be found")
	}
}

func createCRWithIngressinWatchednamespace(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	namespace, err := ctx.GetNamespace()
	helpers.AssertNoError(t, err)

	cr := &v1alpha1.Install{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Install",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipelines-install",
			Namespace: namespace,
		},
		Spec:  v1alpha1.InstallSpec{
			Hostname:  "ingress.test.com",
		},
	}

	cleanup := &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       5 * time.Second,
		RetryInterval: 1 * time.Second,
	}

	f := framework.Global
	err = f.Client.Create(context.TODO(), cr, cleanup)
	helpers.AssertNoError(t, err)

	err = e2eutil.WaitForDeployment(
		t,
		f.KubeClient,
		namespace,
		"tekton-pipelines-controller",
		1,
		config.APIRetry,
		config.APITimeout,
	)
	helpers.AssertNoError(t, err)

	err = e2eutil.WaitForDeployment(
		t,
		f.KubeClient,
		namespace,
		"tekton-pipelines-webhook",
		1,
		config.APIRetry,
		config.APITimeout,
	)
	helpers.AssertNoError(t, err)

	ingress, err := f.KubeClient.ExtensionsV1beta1().Ingresses(namespace).Get("tekton-dashboard", metav1.GetOptions{})
	helpers.AssertNoError(t, err)
	if ingress.Spec.Rules[0].Host != "ingress.test.com" {
		t.Error("Ingress host is not correct")
	}

	controller, err := f.KubeClient.AppsV1().Deployments(namespace).Get("tekton-pipelines-controller", metav1.GetOptions{})
	envs := controller.Spec.Template.Spec.Containers[0].Env
	found := false
	for _, env := range envs {
		if env.Name == "HOST_NAME" && env.Value == "ingress.test.com" {
			found = true
		}
	}
	if !found {
		t.Error("Environment variable is not set")
	}

	webhook, err := f.KubeClient.AppsV1().Deployments(namespace).Get("tekton-pipelines-webhook", metav1.GetOptions{})
	envs = webhook.Spec.Template.Spec.Containers[0].Env
	found = false
	for _, env := range envs {
		if env.Name == "HOST_NAME" && env.Value == "ingress.test.com" {
			found = true
		}
	}
	if !found {
		t.Error("Environment variable is not set")
	}
}
