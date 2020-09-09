package testsuites

import (
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/test"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/flag"
	"github.com/tektoncd/operator/test/helpers"
)

// ValidateAutoInstall creates an instance of install.tekton.dev
// and checks whether openshift pipelines deployment are created
func ValidateAutoInstall(t *testing.T) {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	cr := helpers.WaitForClusterCR(t, flag.ClusterCRName)

	helpers.ValidatePipelineSetup(t, cr,
		flag.PipelineControllerName,
		flag.PipelineWebhookName)

	helpers.ValidatePipelineSetup(t, cr,
		flag.TriggerControllerName,
		flag.TriggerWebhookName)
	err := helpers.WaitForClusterCRStatus(t, flag.ClusterCRName, op.InstalledStatus)
	helpers.AssertNoError(t, err)
}

// ValidateDeletion ensures that deleting the cluster CR  deletes the already
// installed tekton pipeline
func ValidateDeletion(t *testing.T) {
	ctx := test.NewContext(t)
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
}
