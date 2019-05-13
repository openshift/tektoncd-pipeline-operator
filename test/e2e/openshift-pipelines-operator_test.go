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

var (
	deploymentRetry    = 5 * time.Second
	deploymentTimeout  = 240 * time.Second
	cleanupRetry       = 1 * time.Second
	cleanupTimeout     = 5 * time.Second
	operatorDeployment = "openshift-pipelines-operator"
	pipelinesNamespace = "tekton-pipelines"
)

func TestPipelineOperator(t *testing.T) {
	piplnOptrList := &v1alpha1.InstallList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Install",
			APIVersion: "tekton.dev/v1alpha1",
		},
	}

	err := framework.AddToFrameworkScheme(apis.AddToScheme, piplnOptrList)
	assertNoError(t, err)

	t.Run("pipeline-operator can install pipelines", pipelineOperator)
}

func pipelineOperator(t *testing.T) {

	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

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
}

func createCR(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}

	exampleInstallCR := &v1alpha1.Install{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Install",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipelines-install",
			Namespace: namespace,
		},
	}

	cleanupOptions := &framework.CleanupOptions{
		TestContext:   ctx,
		Timeout:       5 * time.Second,
		RetryInterval: 1 * time.Second,
	}

	err = f.Client.Create(context.TODO(), exampleInstallCR, cleanupOptions)
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
		return fmt.Errorf("error in tekton-pipelines-controller deployment: %v", err)
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
		return fmt.Errorf("error in tekton-pipelines-webhook deployment: %v", err)
	}
	return nil
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
