package routers

// Common field names used across routers
const (
	fieldError     = "error"
	fieldStatus    = "status"
	fieldEndpoint  = "endpoint"
	fieldType      = "type"
	fieldPath      = "path"
	fieldUsername  = "username"
	fieldTimestamp = "timestamp"
	fieldLimit     = "limit"
	fieldOffset    = "offset"
	fieldTotal     = "total"
)

// Status values
const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
	statusPassed    = "passed"
	statusFailed    = "failed"
	statusValid     = "valid"
)

// HTTP methods
const (
	methodGET = "GET"
)

// Default values
const (
	defaultConfigDir  = "./config"
	defaultStorageDir = "STORAGE_DIR"
)
