package ansible

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest represents the galaxy.yml file content
type Manifest struct {
	Namespace       string            `yaml:"namespace" json:"namespace"`
	Name            string            `yaml:"name" json:"name"`
	Version         string            `yaml:"version" json:"version"`
	Authors         []string          `yaml:"authors" json:"authors"`
	Description     string            `yaml:"description" json:"description"`
	License         []string          `yaml:"license" json:"license"`
	Tags            []string          `yaml:"tags" json:"tags"`
	Dependencies    map[string]string `yaml:"dependencies" json:"dependencies"`
	Repository      string            `yaml:"repository" json:"repository"`
	Documentation   string            `yaml:"documentation" json:"documentation"`
	Homepage        string            `yaml:"homepage" json:"homepage"`
	Issues          string            `yaml:"issues" json:"issues"`
	RequiresAnsible string            `yaml:"requires_ansible" json:"requires_ansible"`
}

// Validate validates the manifest
func (m *Manifest) Validate() error {
	// Required fields
	if m.Namespace == "" {
		return fmt.Errorf("%w: namespace", ErrMissingRequiredField)
	}
	if m.Name == "" {
		return fmt.Errorf("%w: name", ErrMissingRequiredField)
	}
	if m.Version == "" {
		return fmt.Errorf("%w: version", ErrMissingRequiredField)
	}

	// Validate namespace
	if err := ValidateNamespace(m.Namespace); err != nil {
		return err
	}

	// Validate name
	if err := ValidateName(m.Name); err != nil {
		return err
	}

	// Validate version
	if err := ValidateVersion(m.Version); err != nil {
		return err
	}

	// Validate authors (at least one required)
	if len(m.Authors) == 0 {
		return fmt.Errorf("%w: authors", ErrMissingRequiredField)
	}

	// Validate dependencies
	if err := m.ValidateDependencies(); err != nil {
		return err
	}

	return nil
}

// ValidateDependencies validates dependency specifications
func (m *Manifest) ValidateDependencies() error {
	for collection, versionRange := range m.Dependencies {
		// Parse collection name
		parts := strings.Split(collection, ".")
		if len(parts) != 2 {
			return fmt.Errorf("%w: invalid collection name %s", ErrInvalidDependency, collection)
		}

		namespace, name := parts[0], parts[1]

		// Validate namespace and name
		if err := ValidateNamespace(namespace); err != nil {
			return fmt.Errorf("%w: dependency %s", ErrInvalidDependency, collection)
		}
		if err := ValidateName(name); err != nil {
			return fmt.Errorf("%w: dependency %s", ErrInvalidDependency, collection)
		}

		// Validate version range if not "*"
		if versionRange != "*" && versionRange != "" {
			// Try to parse version range
			// For now, just check if it starts with a valid operator
			versionRange = strings.TrimSpace(versionRange)
			validOps := []string{">=", "<=", "==", "!=", "~=", ">", "<"}
			hasValidOp := false
			for _, op := range validOps {
				if strings.HasPrefix(versionRange, op) {
					hasValidOp = true
					break
				}
			}

			if !hasValidOp {
				return fmt.Errorf("%w: invalid version range %s for %s", ErrInvalidDependency, versionRange, collection)
			}
		}
	}

	return nil
}

// ParseManifestFromTarball extracts and parses galaxy.yml from a tarball
func ParseManifestFromTarball(tarballReader io.Reader) (*Manifest, error) {
	// Create gzip reader
	gzr, err := gzip.NewReader(tarballReader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTarballCorrupted, err)
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Look for galaxy.yml in the tarball
	var manifestData []byte
	found := false

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTarballCorrupted, err)
		}

		// Check if this is galaxy.yml
		// It can be at root or in a subdirectory (e.g., namespace-name-version/galaxy.yml)
		if strings.HasSuffix(header.Name, "galaxy.yml") || strings.HasSuffix(header.Name, "galaxy.yaml") {
			manifestData, err = io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("%w: %v", ErrManifestInvalid, err)
			}
			found = true
			break
		}
	}

	if !found {
		return nil, ErrManifestNotFound
	}

	// Parse YAML
	var manifest Manifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrManifestInvalid, err)
	}

	// Validate manifest
	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// ParseManifest parses galaxy.yml from YAML bytes
func ParseManifest(data []byte) (*Manifest, error) {
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrManifestInvalid, err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// ToYAML converts the manifest to YAML format
func (m *Manifest) ToYAML() ([]byte, error) {
	return yaml.Marshal(m)
}
