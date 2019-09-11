package helpers

import (
	"context"
	"testing"

	versioned "github.com/coreos/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	ctrllercfg "github.com/tektoncd/operator/pkg/controller/config"
	"github.com/tektoncd/operator/test/config"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AssertNoError confirms the error returned is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func CreateConfigCR(t *testing.T, ctx *test.TestCtx, name, targetNs string) {
	cr := &op.Config{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: op.ConfigSpec {
		 TargetNamespace: targetNs,
		},
	}

	err := test.Global.Client.Create(context.TODO(), cr, &test.CleanupOptions{TestContext: ctx,})
	AssertNoError(t, err)
}

// WaitForDeploymentDeletion checks to see if a given deployment is deleted
// the function returns an error if the given deployment is not deleted within the timeout
func WaitForDeploymentDeletion(t *testing.T, namespace, name string) error {
	t.Helper()

	err := wait.Poll(config.APIRetry, config.APITimeout, func() (bool, error) {
		kc := test.Global.KubeClient
		_, err := kc.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsGone(err) || apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}

		t.Logf("Waiting for deletion of %s deployment\n", name)
		return false, nil
	})
	if err == nil {
		t.Logf("%s Deployment deleted\n", name)
	}
	return err
}

func WaitForClusterCR(t *testing.T, name string, completesReconcile bool) *op.Config {
	t.Helper()

	objKey := types.NamespacedName{Name: name}
	cr := &op.Config{}

	err := wait.Poll(config.APIRetry, config.APITimeout, func() (bool, error) {
		err := test.Global.Client.Get(context.TODO(), objKey, cr)
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s cr\n", name)
				return false, nil
			}
			return false, err
		}

		if completesReconcile {
			if cr.IsUpToDateWith(ctrllercfg.TektonVersion) {
				return true, nil
			}

			return false, nil
		}

		if cr.IsMarkedInValid(ctrllercfg.UnknownVersion) {
			return true, nil
		}

		return false, nil
	})

	AssertNoError(t, err)
	return cr
}

func DeleteClusterCR(t *testing.T, name string) {
	t.Helper()

	// ensure object exists before deletion
	objKey := types.NamespacedName{Name: name}
	cr := &op.Config{}
	err := test.Global.Client.Get(context.TODO(), objKey, cr)
	if err != nil {
		t.Logf("Failed to find cluster CR: %s : %s\n", name, err)
	}
	AssertNoError(t, err)

	err = wait.Poll(config.APIRetry, config.APITimeout, func() (bool, error) {
		err := test.Global.Client.Delete(context.TODO(), cr)
		if err != nil {
			t.Logf("Deletion of CR %s failed %s \n", name, err)
			return false, err
		}

		return true, nil
	})

	AssertNoError(t, err)
}

func ValidatePipelineSetup(t *testing.T, cr *op.Config, deployments ...string) {
	t.Helper()

	kc := test.Global.KubeClient
	ns := cr.Spec.TargetNamespace

	for _, d := range deployments {
		err := e2eutil.WaitForDeployment(
			t, kc, ns,
			d,
			1,
			config.APIRetry,
			config.APITimeout,
		)
		AssertNoError(t, err)
	}
}

func ValidatePipelineCleanup(t *testing.T, cr *op.Config, deployments ...string) {
	t.Helper()

	ns := cr.Spec.TargetNamespace
	for _, d := range deployments {
		err := WaitForDeploymentDeletion(t, ns, d)
		AssertNoError(t, err)
	}
}

func ValidateMetricsSetup(t *testing.T, cr *op.Config) {
	t.Helper()
	validateServiceMonitor(t, cr)
	validateRBACRoleForMetrics(t, cr)
	validateRBACRoleBindingForMetrics(t, cr)
}

func validateServiceMonitor(t *testing.T, cr *op.Config) {
	t.Helper()

	monClient, err := versioned.NewForConfig(test.Global.KubeConfig)
	if err != nil {
		t.Errorf("Failed to get the ServiceMonitor client %v \n", err.Error())
		return
	}
	monitors, err := monClient.ServiceMonitors(cr.Spec.TargetNamespace).List(metav1.ListOptions{})
	if err != nil {
		t.Errorf("Failed to get the ServiceMonitor %v \n", err.Error())
		return
	}

	for _, sm := range monitors.Items {
		if len(sm.OwnerReferences) == 0 {
			continue
		}
		if matchOwner(sm.OwnerReferences, cr) {
			return
		}
	}

	t.Fatalf("Expected service monitor with owner %s of kind %s but its not found\n", cr.Name, cr.Kind)
}

func validateRBACRoleForMetrics(t *testing.T, cr *op.Config) {
	t.Helper()

	roles := rbacv1.RoleList{}
	listOpts := client.ListOptions{
		Namespace: cr.Spec.TargetNamespace,
	}
	err := test.Global.Client.List(context.TODO(), &listOpts, &roles)
	if err != nil {
		t.Errorf("Failed to get RBAC roles %v", err.Error())
		return
	}

	for _, r := range roles.Items {
		if len(r.OwnerReferences) == 0 {
			continue
		}

		if(matchOwner(r.OwnerReferences, cr)) {
			return
		}
	}
	t.Fatalf("Expected RBAC role with owner %s but its not found\n", cr.Name)
}

func validateRBACRoleBindingForMetrics(t *testing.T, cr *op.Config) {
	t.Helper()

	roleBindings := rbacv1.RoleBindingList{}
	listOpts := client.ListOptions{
		Namespace: cr.Spec.TargetNamespace,
	}
	err := test.Global.Client.List(context.TODO(), &listOpts, &roleBindings)
	if err != nil {
		t.Errorf("Failed to get RBAC role bindings %v", err.Error())
		return
	}

	for _, r := range roleBindings.Items {
		if len(r.OwnerReferences) == 0 {
			continue
		}

		if(matchOwner(r.OwnerReferences, cr)) {
			return
		}
	}
	t.Errorf("Expected RBAC role bindings with owner %s but its not found\n", cr.Name)
}

func matchOwner(owners []metav1.OwnerReference, cr *op.Config) bool {
	for _, o := range owners {
		if o.Name == cr.Name{
			return true
		}
	}

	return false
}
