package helpers

import (
	"context"
	"fmt"
	"testing"

	secv1 "github.com/openshift/api/security/v1"
	secclient "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/test/config"
	"k8s.io/apimachinery/pkg/types"
)

// AssertNoError confirms the error returned is nil
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// WaitForDeploymentDeletion checks to see if a given deployment is deleted
// the function returns an error if the given deployment is not deleted within the timeout
func WaitForDeploymentDeletion(t *testing.T, namespace, name string) error {
	t.Helper()

	err := wait.Poll(config.APIRetry, config.APITimeout, func() (bool, error) {
		kc := test.Global.KubeClient
		_, err := kc.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{})
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

func WaitForClusterCR(t *testing.T, name string) *op.Config {
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

		return true, nil
	})

	AssertNoError(t, err)
	return cr
}

func WaitForServiceAccount(t *testing.T, ns, targetSA string) *corev1.ServiceAccount {
	t.Helper()

	//objKey := types.NamespacedName{Name: name}
	ret := &corev1.ServiceAccount{}

	err := wait.Poll(config.APIRetry, config.APITimeout, func() (bool, error) {
		saList, err := test.Global.KubeClient.CoreV1().ServiceAccounts(ns).List(metav1.ListOptions{})
		for _, sa := range saList.Items {
			if sa.Name == targetSA {
				ret = &sa
				return true, nil
			}
		}
		return false, err
	})

	AssertNoError(t, err)
	return ret
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
func ValidateSCC(t *testing.T, ns, sa, sccName string) {
	err := wait.Poll(config.APIRetry, config.APITimeout, func() (bool, error) {
		scc, err := GetPrivilegedSCC(sccName)
		if err != nil {
			t.Logf("failed to get privileged scc: %s \n", err)
			return false, err
		}
		t.Log("scc", scc.Users)
		ctrlSA := fmt.Sprintf("system:serviceaccount:%s:%s", ns, sa)
		return inList(scc.Users, ctrlSA), nil
	})
	AssertNoError(t, err)
}

func ValidateSCCDeleted(t *testing.T, ns, sa, sccName string) {
	t.Helper()
	err := wait.Poll(config.APIRetry, config.APITimeout, func() (bool, error) {
		_, err := GetPrivilegedSCC(sccName)
		if err != nil {
			if apierrors.IsGone(err) || apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		t.Logf("Waiting for deletion of %s SCC\n", sccName)
		return false, nil
	})
	if err == nil {
		t.Logf("%s SCC deleted\n", sccName)
	}
	AssertNoError(t, err)
}

func inList(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

func ValidatePipelineSetup(t *testing.T, cr *op.Config, sa string, deployments ...string) {
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

func GetPrivilegedSCC(sccName string) (*secv1.SecurityContextConstraints, error) {
	sec, err := secclient.NewForConfig(test.Global.KubeConfig)
	if err != nil {
		return nil, err
	}
	return sec.SecurityContextConstraints().Get(sccName, metav1.GetOptions{})
}

func ValidatePipelineCleanup(t *testing.T, cr *op.Config, deployments ...string) {
	t.Helper()

	ns := cr.Spec.TargetNamespace
	for _, d := range deployments {
		err := WaitForDeploymentDeletion(t, ns, d)
		AssertNoError(t, err)
	}
}
