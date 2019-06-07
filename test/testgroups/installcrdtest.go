package testgroups

import (
	"testing"

	"github.com/openshift/tektoncd-pipeline-operator/test/config"
	"github.com/openshift/tektoncd-pipeline-operator/test/helpers"
	"github.com/openshift/tektoncd-pipeline-operator/test/testsuites"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
)

// InstallCRDTestGroup is the test group for testing 'Install CRD'
func InstallCRDTestGroup(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	err := deployOperator(t, ctx)
	defer ctx.Cleanup()
	helpers.AssertNoError(t, err)

	t.Run("create-install-cr_installs-pipelines", testsuites.CreateInstallCR)
	t.Run("delete-install-cr_uninstalls-pipelines", testsuites.DeleteInstallCR)
}

func deployOperator(t *testing.T, ctx *framework.TestCtx) error {

	err := ctx.InitializeClusterResources(
		&framework.CleanupOptions{
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

	f := framework.Global
	err = e2eutil.WaitForOperatorDeployment(
		t,
		f.KubeClient,
		namespace,
		config.TestOperatorName,
		1,
		config.APIRetry,
		config.APITimeout,
	)
	if err != nil {
		return err
	}
	return nil
}
