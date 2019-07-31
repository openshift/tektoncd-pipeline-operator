package testgroups

import (
	"testing"

	"github.com/tektoncd/operator/test/helpers"
	"github.com/tektoncd/operator/test/testsuites"

	"github.com/operator-framework/operator-sdk/pkg/test"
)

// ImageReplacement is the test group for image replacement for tekton-pipeline
func ImageReplacement(t *testing.T) {
	ctx := test.NewTestCtx(t)
	defer ctx.Cleanup()

	err := deployOperator(t, ctx)
	helpers.AssertNoError(t, err)

	t.Run("create-cr", testsuites.ValidateManualInstall)
}
