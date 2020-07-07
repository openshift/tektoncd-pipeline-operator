package addons

import (
	"fmt"
	"strings"
	"testing"

	mf "github.com/manifestival/manifestival"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"gotest.tools/golden"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreatePipeline(t *testing.T) {
	t.Run("pipeline template generation", func(t *testing.T) {
		var (
			configName = "cluster"
			namespace  = "openshift-pipelines"
		)

		config := newConfig(configName, namespace)
		cl := feedConfigMock(config)

		template, err := mf.NewManifest("testdata/pipelinetemplate.yaml")
		assertNoEror(t, err)

		manifs, err := CreatePipelines(template, cl)
		assertNoEror(t, err)

		for _, m := range manifs.Resources() {
			jsonPipeline, err := m.MarshalJSON()
			assertNoEror(t, err)
			golden.Assert(t, string(jsonPipeline), strings.ReplaceAll(fmt.Sprintf("%s.golden", m.GetName()), "/", "-"))
		}
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

func feedConfigMock(config *op.Config) client.Client {
	objs := []runtime.Object{config}

	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(op.SchemeGroupVersion, config)

	// Create a fake client to mock API calls.
	return fake.NewFakeClientWithScheme(s, objs...)
}

func assertNoEror(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Errorf("assertion failed; expected no error %v", err)
	}
}
