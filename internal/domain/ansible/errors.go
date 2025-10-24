package ansible

import "errors"

// Domain errors
var (
	// Collection validation errors
	ErrInvalidNamespace     = errors.New("invalid namespace: must match ^[a-z0-9_]+$")
	ErrInvalidName          = errors.New("invalid collection name: must match ^[a-z0-9_]+$")
	ErrInvalidVersion       = errors.New("invalid version: must follow semantic versioning")
	ErrNamespaceTooLong     = errors.New("namespace too long: maximum 64 characters")
	ErrNameTooLong          = errors.New("collection name too long: maximum 64 characters")
	ErrReservedNamespace    = errors.New("reserved namespace: cannot use 'ansible' or 'galaxy'")

	// Manifest errors
	ErrManifestNotFound     = errors.New("galaxy.yml not found in tarball")
	ErrManifestInvalid      = errors.New("galaxy.yml is invalid or malformed")
	ErrMissingRequiredField = errors.New("missing required field in galaxy.yml")
	ErrInvalidDependency    = errors.New("invalid dependency specification")

	// Tarball validation errors
	ErrTarballTooLarge      = errors.New("tarball exceeds maximum size limit")
	ErrTarballCorrupted     = errors.New("tarball is corrupted or unreadable")
	ErrInvalidTarballPath   = errors.New("tarball contains invalid or dangerous paths")
	ErrForbiddenFile        = errors.New("tarball contains forbidden files")

	// Version errors
	ErrVersionMismatch      = errors.New("version in filename does not match galaxy.yml")
	ErrVersionAlreadyExists = errors.New("collection version already exists")
	ErrVersionNotFound      = errors.New("collection version not found")
	ErrInvalidVersionRange  = errors.New("invalid version range specification")
)
