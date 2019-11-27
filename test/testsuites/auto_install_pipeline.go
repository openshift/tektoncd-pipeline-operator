package testsuites

import (
	"testing"

	"github.com/tektoncd/operator/pkg/flag"

	"github.com/operator-framework/operator-sdk/pkg/test"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/test/helpers"
)

// ValidateAutoInstall creates an instance of install.tekton.dev
// and checks whether openshift pipelines deployment are created
func ValidateAutoInstall(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	cr := helpers.WaitForClusterCR(t, flag.ClusterCRName)

	helpers.ValidatePipelineSetup(t, cr,
		flag.PipelineControllerName,
		flag.PipelineWebhookName)

	helpers.ValidatePipelineSetup(t, cr,
		flag.TriggerControllerName,
		flag.TriggerWebhookName)

	helpers.ValidateSCCAdded(t, cr.Spec.TargetNamespace, flag.PipelineControllerName)

	cr = helpers.WaitForClusterCR(t, flag.ClusterCRName)
	if code := cr.Status.Conditions[0].Code; code != op.InstalledStatus {
		t.Errorf("Expected code to be %s but got %s", op.InstalledStatus, code)
	}

}

// ValidateDeletion ensures that deleting the cluster CR  deletes the already
// installed tekton pipeline
func ValidateDeletion(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	cr := helpers.WaitForClusterCR(t, flag.ClusterCRName)

	helpers.ValidatePipelineSetup(t, cr,
		flag.PipelineControllerName,
		flag.PipelineWebhookName)

	helpers.ValidatePipelineSetup(t, cr,
		flag.TriggerControllerName,
		flag.TriggerWebhookName)

	helpers.DeleteClusterCR(t, flag.ClusterCRName)

	helpers.ValidatePipelineCleanup(t, cr,
		flag.PipelineControllerName,
		flag.PipelineWebhookName)

	helpers.ValidatePipelineCleanup(t, cr,
		flag.TriggerControllerName,
		flag.TriggerWebhookName)

	helpers.ValidateSCCRemoved(t, cr.Spec.TargetNamespace, flag.PipelineControllerName)
}
