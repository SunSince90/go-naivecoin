package controllers

import (
	"fmt"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewControllerManager creates the pod controller and returns its manager so
// that it could be started.
func NewControllerManager() (manager.Manager, error) {
	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		return nil, fmt.Errorf("could not get namespace from environment variable")
	}

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	if err != nil {
		return nil, err
	}

	return mgr, nil
}
