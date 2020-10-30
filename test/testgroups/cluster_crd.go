package testgroups

import (
	"testing"

	"github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/tektoncd/operator/test/config"
	"github.com/tektoncd/operator/test/helpers"
	"github.com/tektoncd/operator/test/testsuites"
)

// ClusterCRD is the test group for testing config.operator.tekton.dev CRD
func ClusterCRD(t *testing.T) {
	ctx := test.NewContext(t)
	defer ctx.Cleanup()

	err := deployOperator(t, ctx)
	helpers.AssertNoError(t, err)

	t.Run("auto-installs-pipelines", testsuites.ValidateAutoInstall)
	t.Run("auto-create-sa", testsuites.ValidateDefaultSA)
	t.Run("validate-anyuid-clusterrole", testsuites.ValidateClusterRole)
	t.Run("validate-anyuid-rolebinding", testsuites.ValidateSCCRoleBinding)
	t.Run("delete-pipelines", testsuites.ValidateDeletion)
}

func deployOperator(t *testing.T, ctx *test.Context) error {
	err := ctx.InitializeClusterResources(
		&test.CleanupOptions{
			TestContext:   ctx,
			Timeout:       config.CleanupTimeout,
			RetryInterval: config.CleanupRetry,
		},
	)
	if err != nil {
		return err
	}

	namespace, err := ctx.GetNamespace()
	if err != nil {
		return err
	}

	return e2eutil.WaitForOperatorDeployment(
		t,
		test.Global.KubeClient,
		namespace,
		config.TestOperatorName,
		1,
		config.APIRetry,
		config.APITimeout,
	)
}
