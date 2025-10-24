package ansible

import (
	"proxynd/internal/config"
	"proxynd/internal/ports"
)

// AnsibleHandler handles Ansible Galaxy v3 API requests
type AnsibleHandler struct {
	hostedDriver ports.HostedRepository
	config       *config.AnsibleRepositoryConfig
	logger       ports.Logger
}

// NewAnsibleHandler creates a new Ansible handler
func NewAnsibleHandler(
	hostedDriver ports.HostedRepository,
	cfg *config.AnsibleRepositoryConfig,
	logger ports.Logger,
) *AnsibleHandler {
	return &AnsibleHandler{
		hostedDriver: hostedDriver,
		config:       cfg,
		logger:       logger,
	}
}
