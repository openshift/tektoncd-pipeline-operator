package testsuites

import (
	"testing"

	op "github.com/openshift/tektoncd-pipeline-operator/pkg/apis/operator/v1alpha1"
	"github.com/openshift/tektoncd-pipeline-operator/pkg/controller/config"
	"github.com/openshift/tektoncd-pipeline-operator/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
)

// ValidateManualInstall creates an instance of config.tekton.dev
// and checks whether openshift pipelines deployment are created
// The CR will include image replacement attribute and on purpose to use a wrong image, then check if the POD is failed
func ValidateManualInstall(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	helpers.DeleteClusterCR(t, config.ClusterCRName)
	helpers.CreateCR(t, config.ClusterCRName, config.DefaultTargetNs, ctx)
	cr := helpers.WaitForClusterCR(t, config.ClusterCRName)
	helpers.ValidatePipelineSetup(t, cr,
		config.PipelineWebhookName)
	helpers.ValidatePipelineFailure(t, cr,
		config.PipelineControllerName)

	cr = helpers.WaitForClusterCR(t, config.ClusterCRName)
	if code := cr.Status.Conditions[0].Code; code != op.InstalledStatus {
		t.Errorf("Expected code to be %s but got %s", op.InstalledStatus, code)
	}

}
