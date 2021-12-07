package controllers

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate mockgen -package=controllers -source=external_interfaces.go -destination=external_interfaces_mock.go

// K8sClient for mock
type K8sClient interface {
	client.Client
}

// Logger for mock
type Logger interface {
	logr.Logger
}
