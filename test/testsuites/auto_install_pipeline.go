package testsuites

import (
	"testing"

	"github.com/openshift/tektoncd-pipeline-operator/test/config"
	"github.com/openshift/tektoncd-pipeline-operator/test/helpers"
	"github.com/operator-framework/operator-sdk/pkg/test"
)

// ValidateAutoInstall creates an instance of install.tekton.dev
// and checks whether openshift pipelines deployment are created
func ValidateAutoInstall(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	cr := helpers.WaitForClusterCR(t, config.ClusterCR)
	helpers.ValidatePipelineSetup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)
}

// ValidateDeletion ensures that deleting the cluster CR  deletes the already
// installed tekton pipeline
func ValidateDeletion(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	cr := helpers.WaitForClusterCR(t, config.ClusterCR)
	helpers.ValidatePipelineSetup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)

	helpers.DeleteClusterCR(t, config.ClusterCR)

	helpers.ValidatePipelineCleanup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)
}
