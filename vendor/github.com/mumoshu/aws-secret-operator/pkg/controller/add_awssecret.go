package controller

import (
	"github.com/mumoshu/aws-secret-operator/pkg/controller/awssecret"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, awssecret.Add)
}
