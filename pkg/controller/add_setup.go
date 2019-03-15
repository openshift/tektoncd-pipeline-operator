package controller

import (
	"github.com/openshift-cloud-functions/tektoncd-operator/pkg/controller/setup"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, setup.Add)
}
