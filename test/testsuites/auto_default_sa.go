package testsuites

import (
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/test"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/flag"
	"github.com/tektoncd/operator/test/helpers"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidateDefaultSA validates that tekton controller creates
// a Default pipelines SA for existing and new namespaces
func ValidateDefaultSA(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	// ensure the controllers are running and pipeline is installed
	err := helpers.WaitForClusterCRStatus(t, flag.ClusterCRName, op.InstalledStatus)
	helpers.AssertNoError(t, err)

	helpers.WaitForServiceAccount(t, "default", flag.DefaultSA)

	// Create a namespace
	newNs := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foobar"}}
	ns, err := test.Global.KubeClient.CoreV1().Namespaces().Create(&newNs)
	// cleanup
	defer func() {
		// lint complains about return err ignored
		_ = test.Global.KubeClient.CoreV1().Namespaces().Delete(ns.Name, &metav1.DeleteOptions{})
	}()
	if !apierrors.IsAlreadyExists(err) {
		helpers.AssertNoError(t, err)
	}
	helpers.WaitForServiceAccount(t, ns.Name, flag.DefaultSA)
}
