package testsuites

import (
	"testing"

	"github.com/tektoncd/operator/pkg/controller/flags"

	"github.com/operator-framework/operator-sdk/pkg/test"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/test/helpers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidateDefaultSA validates that tekton controller creates
// a Default pipelines SA for existing and new namespaces
func ValidateDefaultSA(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// ensure the controllers are running and pipeline is installed
	cr := helpers.WaitForClusterCR(t, flags.ClusterCRName)
	if code := cr.Status.Conditions[0].Code; code != op.InstalledStatus {
		t.Errorf("Expected code to be %s but got %s", op.InstalledStatus, code)
	}

	helpers.WaitForServiceAccount(t, "default", flags.DefaultSA)

	// Create a namespace
	newNs := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foobar"}}
	ns, err := test.Global.KubeClient.CoreV1().Namespaces().Create(&newNs)
	helpers.AssertNoError(t, err)

	// cleanup
	defer func() {
		// lint complains about return err ignored
		_ = test.Global.KubeClient.CoreV1().Namespaces().Delete(ns.Name, &metav1.DeleteOptions{})
	}()

	helpers.WaitForServiceAccount(t, ns.Name, flags.DefaultSA)
}
