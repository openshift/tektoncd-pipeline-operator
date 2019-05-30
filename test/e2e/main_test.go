package e2e

import (
	"testing"

	"github.com/openshift/tektoncd-pipeline-operator/pkg/apis"
	"github.com/openshift/tektoncd-pipeline-operator/pkg/apis/tekton/v1alpha1"
	"github.com/openshift/tektoncd-pipeline-operator/test/testgroups"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMain(m *testing.M) {
	framework.MainEntry(m)
}

func TestPipelineOperator(t *testing.T) {
	initTestingFramework(t)

	//Run test groups
	t.Run("install-crd", testgroups.InstallCRDTestGroup)
}

func initTestingFramework(t *testing.T) {
	installList := &v1alpha1.InstallList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Install",
			APIVersion: "tekton.dev/v1alpha1",
		},
	}

	err := framework.AddToFrameworkScheme(apis.AddToScheme, installList)
	if err != nil {
		t.Fatalf(
			"failed to add 'tekton.dev/v1alpha1 Install' scheme to test framework: %v",
			err,
		)
	}
}
