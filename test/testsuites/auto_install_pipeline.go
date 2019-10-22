package testsuites

import (
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/test"
	op "github.com/tektoncd/operator/pkg/apis/operator/v1alpha1"
	"github.com/tektoncd/operator/pkg/controller/config"
	"github.com/tektoncd/operator/test/helpers"
	"gotest.tools/icmd"
)

// ValidateAutoInstall creates an instance of install.tekton.dev
// and checks whether openshift pipelines deployment are created
func ValidateAutoInstall(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	cr := helpers.WaitForClusterCR(t, config.ClusterCRName)
	helpers.ValidatePipelineSetup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)

	helpers.ValidateSCCAdded(t, cr.Spec.TargetNamespace, config.PipelineControllerName)

	cr = helpers.WaitForClusterCR(t, config.ClusterCRName)
	if code := cr.Status.Conditions[0].Code; code != op.InstalledStatus {
		t.Errorf("Expected code to be %s but got %s", op.InstalledStatus, code)
	}

}

// ValidateDeletion ensures that deleting the cluster CR  deletes the already
// installed tekton pipeline
func ValidateDeletion(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	cr := helpers.WaitForClusterCR(t, config.ClusterCRName)
	helpers.ValidatePipelineSetup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)

	helpers.DeleteClusterCR(t, config.ClusterCRName)

	helpers.ValidatePipelineCleanup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)
	helpers.ValidateSCCRemoved(t, cr.Spec.TargetNamespace, config.PipelineControllerName)
}

//Validate TaskRun install on random namespace and test for its completion
func ValidateE2EPipelines(t *testing.T) {
	t.Helper()
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()
	defer DeletePipelineResources(t)

	cr := helpers.WaitForClusterCR(t, config.ClusterCRName)
	helpers.ValidatePipelineSetup(t, cr,
		config.PipelineControllerName,
		config.PipelineWebhookName)

	helpers.ValidateSCCAdded(t, cr.Spec.TargetNamespace, config.PipelineControllerName)

	cr = helpers.WaitForClusterCR(t, config.ClusterCRName)
	if code := cr.Status.Conditions[0].Code; code != op.InstalledStatus {
		t.Errorf("Expected code to be %s but got %s", op.InstalledStatus, code)
	}
	namespace := "tektoncd"
	CreateNamespaceAndSetContext(t, namespace)

	// Create sample Taskrun into any of the namespace
	run := Prepare(t)

	res := icmd.RunCmd(run("apply", "-f", "./test/resources/task-volume.yaml", "-n", namespace))

	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      icmd.None,
	})

	RunE2ETests(t)

}

func CreateNamespaceAndSetContext(t *testing.T, namespace string) {
	t.Helper()
	run := Prepare(t)
	res_namespace := icmd.RunCmd(run("create", "ns", namespace))

	res_namespace.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      icmd.None,
	})

	t.Logf("Created namespace %s successfully..", namespace)

	res_setCurrentContext := icmd.RunCmd(run("config", "set-context", icmd.RunCmd(run("config", "current-context")).Stdout(), "--namespace", namespace))

	res_setCurrentContext.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      icmd.None,
	})
}

func RunE2ETests(t *testing.T) {
	t.Helper()
	t.Logf("Running E2E tests..")
	res_e2e := icmd.RunCmd(icmd.Command("./test/helpers/wait_test.sh"))

	res_e2e.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      icmd.None,
	})

	t.Logf(res_e2e.Stdout())
}

func Prepare(t *testing.T) func(args ...string) icmd.Cmd {

	run := func(args ...string) icmd.Cmd {
		return icmd.Command("kubectl", append([]string{}, args...)...)
	}
	return run
}

func DeletePipelineResources(t *testing.T) {
	run := Prepare(t)

	res := icmd.RunCmd(run("delete", "ns", "tektoncd"))

	res.Assert(t, icmd.Expected{
		ExitCode: 0,
		Err:      icmd.None,
	})
}
