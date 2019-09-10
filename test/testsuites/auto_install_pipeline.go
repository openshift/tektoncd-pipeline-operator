package testsuites

import (
	"testing"
	
	"github.com/operator-framework/operator-sdk/pkg/test"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/controller/config"
	"github.com/tektoncd/operator/test/helpers"
)

// ValidateAutoInstall creates an instance of install.tekton.dev
// and checks whether openshift pipelines deployment are created
func ValidateAutoInstall(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	cr := helpers.WaitForClusterCR(t, config.ClusterCRName, true)

	if cond := cr.Status.Conditions[0]; cond.Code != op.ReadyStatus {
		t.Errorf("Expected code is %s but got %s, error status %s", op.ReadyStatus, cond.Code, cond.Details)
	}

	helpers.ValidatePipelineSetup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)
}

// ValidateDeletion ensures that deleting the cluster CR  deletes the already
// installed tekton pipeline
func ValidateDeletion(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	cr := helpers.WaitForClusterCR(t, config.ClusterCRName, true)

	helpers.ValidatePipelineSetup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)

	helpers.DeleteClusterCR(t, config.ClusterCRName)

	helpers.ValidatePipelineCleanup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)
}

// CheckInvalidConfig creates an instance of install.tekton.dev
// and checks whether openshift pipelines deployment are created
func CheckInvalidConfig(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	helpers.CreateConfigCR(t, ctx, "random", "openshift-pipelines")
	cr := helpers.WaitForClusterCR(t, "random", false)

	if cond := cr.Status.Conditions[0]; cond.Code != op.ErrorStatus && cond.Version != "unknown" {
		t.Errorf("Expected code is %s but got %s and version %s but got %s", op.ErrorStatus, cond.Code, "unknow", cond.Version)
	}
}
