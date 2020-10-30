package testsuites

import (
	"context"
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/test"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/flag"
	"github.com/tektoncd/operator/test/helpers"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ValidateDefaultSA validates that tekton controller creates
// a Default pipelines SA for existing and new namespaces
func ValidateDefaultSA(t *testing.T) {
	ctx := test.NewContext(t)
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

// ValidateSCCRoleBinding validates that tekton controller creates
// the cluster rolebinding for anyuid
func ValidateSCCRoleBinding(t *testing.T) {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	// ensure the controllers are running and pipeline is installed
	err := helpers.WaitForClusterCRStatus(t, flag.ClusterCRName, op.InstalledStatus)
	helpers.AssertNoError(t, err)

	helpers.WaitForRolebinding(t, "default", flag.PipelineAnyuid)

	// Create a namespace
	newNs := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foobar-rb-anyuid"}}
	ns, err := test.Global.KubeClient.CoreV1().Namespaces().Create(&newNs)
	// cleanup
	defer func() {
		// lint complains about return err ignored
		_ = test.Global.KubeClient.CoreV1().Namespaces().Delete(ns.Name, &metav1.DeleteOptions{})
	}()
	if !apierrors.IsAlreadyExists(err) {
		helpers.AssertNoError(t, err)
	}
	helpers.WaitForRolebinding(t, ns.Name, flag.PipelineAnyuid)
}

// ValidateClusterRole validates that tekton controller creates
// the cluster role for anyuid
func ValidateClusterRole(t *testing.T) {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	// ensure the controllers are running and pipeline is installed
	err := helpers.WaitForClusterCRStatus(t, flag.ClusterCRName, op.InstalledStatus)
	helpers.AssertNoError(t, err)

	helpers.WaitForClusterRole(t, flag.PipelineAnyuid)

	// Create a namespace
	newNs := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "foobar-crole-anyuid"}}
	ns, err := test.Global.KubeClient.CoreV1().Namespaces().Create(&newNs)
	// cleanup
	defer func() {
		// lint complains about return err ignored
		_ = test.Global.KubeClient.CoreV1().Namespaces().Delete(ns.Name, &metav1.DeleteOptions{})
	}()
	if !apierrors.IsAlreadyExists(err) {
		helpers.AssertNoError(t, err)
	}
	helpers.WaitForClusterRole(t, flag.PipelineAnyuid)
}

// WaitForDowngradedSA waits for the default SA's rolebinding to be deleted.
func WaitForDowngradedSA(t *testing.T) error {

	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	oldForbiddenNamespaces := []string{
		"forbidden-ns-1",
		"forbidden-ns-2",
	}

	newForbiddenNamespaces := []string{
		"forbidden-ns-3",
		"forbidden-ns-4",
	}

	// cleanup all created namespaces
	defer func() {
		for _, ns := range append(oldForbiddenNamespaces, newForbiddenNamespaces...) {
			_ = test.Global.KubeClient.CoreV1().Namespaces().Delete(ns, &metav1.DeleteOptions{})
		}
	}()

	cfg := &op.Config{}
	err := test.Global.Client.Get(context.TODO(), types.NamespacedName{Name: flag.ClusterCRName}, cfg)
	if err != nil {
		return err
	}

	// Ensure namespaces exist before they are added to
	// denylist.

	for _, n := range oldForbiddenNamespaces {
		newNs := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: n}}
		_, err := test.Global.KubeClient.CoreV1().Namespaces().Create(&newNs)
		if err != nil {
			return err
		}
	}

	cfg.Spec.NamespaceExclusions = oldForbiddenNamespaces
	err = test.Global.Client.Update(context.TODO(), cfg)
	if err != nil {
		return err
	}

	for _, n := range oldForbiddenNamespaces {
		err := helpers.WaitForRolebindingDeletion(t, n, flag.PipelineAnyuid)
		if err != nil {
			return err
		}
	}

	// Namespace added to denylist before it was created,
	// Namespaces in the existing denylist taken off list.

	cfg.Spec.NamespaceExclusions = newForbiddenNamespaces
	err = test.Global.Client.Update(context.TODO(), cfg)
	if err != nil {
		return err
	}

	// Old exclusions now being taken off denylist
	// They should have the rolebindings restored.

	for _, n := range oldForbiddenNamespaces {
		helpers.WaitForRolebinding(t, n, flag.PipelineAnyuid)
	}

	// Create the new namespaces, should trigger the rbac controller.
	for _, n := range newForbiddenNamespaces {
		newNs := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: n}}
		_, err = test.Global.KubeClient.CoreV1().Namespaces().Create(&newNs)
		if err != nil {
			return err
		}
	}

	for _, n := range cfg.Spec.NamespaceExclusions {
		err := helpers.WaitForRolebindingDeletion(t, n, flag.PipelineAnyuid)
		if err != nil {
			return err
		}
	}
	return nil
}
