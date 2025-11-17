package enterprise

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"proxynd/internal/domain/enterprise"
	"proxynd/internal/logging"
)

// FixtureLoader loads development fixtures for enterprise API
type FixtureLoader struct {
	fixturesDir string
	logger      logging.Logger
}

// NewFixtureLoader creates a new fixture loader
func NewFixtureLoader(fixturesDir string, logger logging.Logger) *FixtureLoader {
	if logger == nil {
		logger = logging.GetLogger()
	}
	return &FixtureLoader{
		fixturesDir: fixturesDir,
		logger:      logger,
	}
}

// LoadAll loads all development fixtures
func (f *FixtureLoader) LoadAll() error {
	f.logger.Info("Loading development fixtures",
		logging.F("dir", f.fixturesDir))

	// Load roles
	if err := f.loadRoles(); err != nil {
		f.logger.Warn("Failed to load roles fixture",
			logging.F("error", err))
	}

	// Load users
	if err := f.loadUsers(); err != nil {
		f.logger.Warn("Failed to load users fixture",
			logging.F("error", err))
	}

	// Load audit events
	if err := f.loadAuditEvents(); err != nil {
		f.logger.Warn("Failed to load audit events fixture",
			logging.F("error", err))
	}

	// Load alerts
	if err := f.loadAlerts(); err != nil {
		f.logger.Warn("Failed to load alerts fixture",
			logging.F("error", err))
	}

	// Load vulnerabilities
	if err := f.loadVulnerabilities(); err != nil {
		f.logger.Warn("Failed to load vulnerabilities fixture",
			logging.F("error", err))
	}

	f.logger.Info("Development fixtures loaded successfully")
	return nil
}

// loadRoles loads role fixtures
func (f *FixtureLoader) loadRoles() error {
	path := filepath.Join(f.fixturesDir, "roles.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading roles fixture: %w", err)
	}

	var roles []*enterprise.Role
	if err := json.Unmarshal(data, &roles); err != nil {
		return fmt.Errorf("parsing roles fixture: %w", err)
	}

	f.logger.Debug("Loaded roles fixture",
		logging.F("count", len(roles)))
	return nil
}

// loadUsers loads user fixtures
func (f *FixtureLoader) loadUsers() error {
	path := filepath.Join(f.fixturesDir, "users.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading users fixture: %w", err)
	}

	var users []map[string]interface{}
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("parsing users fixture: %w", err)
	}

	f.logger.Debug("Loaded users fixture",
		logging.F("count", len(users)))
	return nil
}

// loadAuditEvents loads audit event fixtures
func (f *FixtureLoader) loadAuditEvents() error {
	path := filepath.Join(f.fixturesDir, "audit_events.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading audit events fixture: %w", err)
	}

	var events []*enterprise.AuditEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return fmt.Errorf("parsing audit events fixture: %w", err)
	}

	f.logger.Debug("Loaded audit events fixture",
		logging.F("count", len(events)))
	return nil
}

// loadAlerts loads alert fixtures
func (f *FixtureLoader) loadAlerts() error {
	path := filepath.Join(f.fixturesDir, "alerts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading alerts fixture: %w", err)
	}

	var alerts []*enterprise.Alert
	if err := json.Unmarshal(data, &alerts); err != nil {
		return fmt.Errorf("parsing alerts fixture: %w", err)
	}

	f.logger.Debug("Loaded alerts fixture",
		logging.F("count", len(alerts)))
	return nil
}

// loadVulnerabilities loads vulnerability fixtures
func (f *FixtureLoader) loadVulnerabilities() error {
	path := filepath.Join(f.fixturesDir, "vulnerabilities.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading vulnerabilities fixture: %w", err)
	}

	var vulns []*enterprise.Vulnerability
	if err := json.Unmarshal(data, &vulns); err != nil {
		return fmt.Errorf("parsing vulnerabilities fixture: %w", err)
	}

	f.logger.Debug("Loaded vulnerabilities fixture",
		logging.F("count", len(vulns)))
	return nil
}

// IsAvailable checks if fixtures directory exists
func (f *FixtureLoader) IsAvailable() bool {
	info, err := os.Stat(f.fixturesDir)
	return err == nil && info.IsDir()
}
