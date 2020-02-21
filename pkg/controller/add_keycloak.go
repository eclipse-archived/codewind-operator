package controller

import (
	"github.com/eclipse/codewind-operator/pkg/controller/keycloak"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, keycloak.Add)
}
