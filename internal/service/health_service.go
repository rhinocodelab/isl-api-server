package service

import (
	"time"
)

// HealthService handles health check logic
type HealthService struct{}

// NewHealthService creates a new health service
func NewHealthService() *HealthService {
	return &HealthService{}
}

// GetHealthStatus returns the current health status
func (h *HealthService) GetHealthStatus() map[string]interface{} {
	return map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "isl-api-server",
		"version":   "1.0.0",
	}
}
