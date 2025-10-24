package ansible

import (
	"regexp"
	"strings"
)

var (
	// Namespace and name must match [a-z0-9_]+
	namespaceRegex = regexp.MustCompile(`^[a-z0-9_]+$`)
	nameRegex      = regexp.MustCompile(`^[a-z0-9_]+$`)

	// Reserved namespaces that cannot be used
	reservedNamespaces = map[string]bool{
		"ansible": true,
		"galaxy":  true,
	}

	// Maximum lengths
	maxNamespaceLength = 64
	maxNameLength      = 64
)

// Collection represents an Ansible collection
type Collection struct {
	Namespace string
	Name      string
	Version   string
	Manifest  *Manifest
	SHA256    string
	Size      int64
}

// FullName returns the full collection name in the format "namespace.name"
func (c *Collection) FullName() string {
	return c.Namespace + "." + c.Name
}

// Filename returns the expected tarball filename
func (c *Collection) Filename() string {
	return c.Namespace + "-" + c.Name + "-" + c.Version + ".tar.gz"
}

// Validate validates the collection metadata
func (c *Collection) Validate() error {
	if err := ValidateNamespace(c.Namespace); err != nil {
		return err
	}

	if err := ValidateName(c.Name); err != nil {
		return err
	}

	if err := ValidateVersion(c.Version); err != nil {
		return err
	}

	// Validate manifest if present
	if c.Manifest != nil {
		if err := c.Manifest.Validate(); err != nil {
			return err
		}

		// Ensure manifest matches collection metadata
		if c.Manifest.Namespace != c.Namespace {
			return ErrManifestInvalid
		}
		if c.Manifest.Name != c.Name {
			return ErrManifestInvalid
		}
		if c.Manifest.Version != c.Version {
			return ErrVersionMismatch
		}
	}

	return nil
}

// ValidateNamespace validates an Ansible namespace
func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return ErrInvalidNamespace
	}

	if len(namespace) > maxNamespaceLength {
		return ErrNamespaceTooLong
	}

	// Check reserved namespaces
	if reservedNamespaces[strings.ToLower(namespace)] {
		return ErrReservedNamespace
	}

	// Check pattern
	if !namespaceRegex.MatchString(namespace) {
		return ErrInvalidNamespace
	}

	return nil
}

// ValidateName validates a collection name
func ValidateName(name string) error {
	if name == "" {
		return ErrInvalidName
	}

	if len(name) > maxNameLength {
		return ErrNameTooLong
	}

	// Check pattern
	if !nameRegex.MatchString(name) {
		return ErrInvalidName
	}

	return nil
}
