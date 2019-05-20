package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	apis "github.com/openshift/tektoncd-pipeline-operator/pkg/apis"
	"github.com/openshift/tektoncd-pipeline-operator/pkg/apis/tekton/v1alpha1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	deploymentRetry    = 5 * time.Second
	deploymentTimeout  = 240 * time.Second
	cleanupRetry       = 1 * time.Second
	cleanupTimeout     = 5 * time.Second
	operatorDeployment = "openshift-pipelines-operator"
	pipelinesNamespace = "tekton-pipelines"
)

func TestPipelineOperator(t *testing.T) {
	installList := &v1alpha1.InstallList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Install",
			APIVersion: "tekton.dev/v1alpha1",
		},
	}

	err := framework.AddToFrameworkScheme(apis.AddToScheme, installList)
	assertNoError(t, err)

	t.Run("pipeline-operator can install pipelines", pipelineOperator)
}

func pipelineOperator(t *testing.T) {

	ctx := framework.NewTestCtx(t)

	err := ctx.InitializeClusterResources(
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       cleanupTimeout,
			RetryInterval: cleanupRetry,
		},
	)
	assertNoError(t, err)

	namespace, err := ctx.GetNamespace()
	assertNoError(t, err)

	f := framework.Global
	err = e2eutil.WaitForOperatorDeployment(
		t,
		f.KubeClient,
		namespace,
		operatorDeployment,
		1,
		deploymentRetry,
		deploymentTimeout,
	)
	assertNoError(t, err)

	err = createCR(t, f, ctx)
	assertNoError(t, err)

	ctx.Cleanup()

	ctx = framework.NewTestCtx(t)
	defer ctx.Cleanup()

	err = ctx.InitializeClusterResources(
		&framework.CleanupOptions{
			TestContext:   ctx,
			Timeout:       cleanupTimeout,
			RetryInterval: cleanupRetry,
		},
	)
	assertNoError(t, err)

	namespace, err = ctx.GetNamespace()
	assertNoError(t, err)

	f = framework.Global
	err = e2eutil.WaitForOperatorDeployment(
		t,
		f.KubeClient,
		namespace,
		operatorDeployment,
		1,
		deploymentRetry,
		deploymentTimeout,
	)
	assertNoError(t, err)

	err = createCRWithAddon(t, f, ctx)
	assertNoError(t, err)
}

func createCR(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

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

	err = f.Client.Create(context.TODO(), cr, cleanup)
	if err != nil {
		return fmt.Errorf("error in creating install CR: %v", err)
	}

	err = e2eutil.WaitForDeployment(
		t,
		f.KubeClient,
		pipelinesNamespace,
		"tekton-pipelines-controller",
		1,
		deploymentRetry,
		deploymentTimeout,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy tekton-pipelines-controller: %s", err)
	}

	err = e2eutil.WaitForDeployment(
		t,
		f.KubeClient,
		pipelinesNamespace,
		"tekton-pipelines-webhook",
		1,
		deploymentRetry,
		deploymentTimeout,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy tekton-pipelines-webhook deployment: %s", err)
	}

	return nil
}
func createCRWithAddon(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	cr := &v1alpha1.Install{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Install",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipelines-install-addon1",
			Namespace: namespace,
		},
		Spec:  v1alpha1.InstallSpec{
			AddOns:     []string {"addon1"},
		},
	}

	cleanup := &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       5 * time.Second,
		RetryInterval: 1 * time.Second,
	}

	err = f.Client.Create(context.TODO(), cr, cleanup)
	if err != nil {
		return fmt.Errorf("error in creating install CR: %v", err)
	}

	err = e2eutil.WaitForDeployment(
		t,
		f.KubeClient,
		pipelinesNamespace,
		"tekton-pipelines-controller",
		1,
		deploymentRetry,
		deploymentTimeout,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy tekton-pipelines-controller: %s", err)
	}

	err = e2eutil.WaitForDeployment(
		t,
		f.KubeClient,
		pipelinesNamespace,
		"tekton-pipelines-webhook",
		1,
		deploymentRetry,
		deploymentTimeout,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy tekton-pipelines-webhook deployment: %s", err)
	}

	err = e2eutil.WaitForDeployment(
		t,
		f.KubeClient,
		pipelinesNamespace,
		"addon1-deployment",
		1,
		deploymentRetry,
		deploymentTimeout,
	)
	if err != nil {
		return fmt.Errorf("failed to deploy addon1-deployment deployment: %s", err)
	}

	return nil
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
