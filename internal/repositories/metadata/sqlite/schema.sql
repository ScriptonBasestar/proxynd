-- ProxyND Metadata Database Schema
-- SQLite-compatible schema for package metadata storage

-- ============================================================================
-- Common Artifacts Table (Content-Addressable Storage)
-- ============================================================================
CREATE TABLE IF NOT EXISTS artifacts (
    sha256 TEXT PRIMARY KEY,
    size INTEGER NOT NULL,
    content_type TEXT,
    ref_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    accessed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (size >= 0),
    CHECK (ref_count >= 0)
);

CREATE INDEX idx_artifacts_ref_count ON artifacts(ref_count);
CREATE INDEX idx_artifacts_accessed_at ON artifacts(accessed_at);

-- ============================================================================
-- Ansible Collections
-- ============================================================================

-- Namespaces (e.g., "community", "ansible")
CREATE TABLE IF NOT EXISTS ansible_namespaces (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    company TEXT,
    avatar_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (length(name) > 0)
);

CREATE INDEX idx_ansible_namespaces_name ON ansible_namespaces(name);

-- Collections (e.g., "community.general")
CREATE TABLE IF NOT EXISTS ansible_collections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    namespace_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    deprecated BOOLEAN NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (namespace_id) REFERENCES ansible_namespaces(id) ON DELETE CASCADE,
    UNIQUE (namespace_id, name),
    CHECK (length(name) > 0)
);

CREATE INDEX idx_ansible_collections_namespace ON ansible_collections(namespace_id);
CREATE INDEX idx_ansible_collections_name ON ansible_collections(name);
CREATE INDEX idx_ansible_collections_deprecated ON ansible_collections(deprecated);

-- Collection Versions (e.g., "community.general" version "1.2.3")
CREATE TABLE IF NOT EXISTS ansible_collection_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    collection_id INTEGER NOT NULL,
    version TEXT NOT NULL,
    artifact_sha256 TEXT NOT NULL,

    -- Metadata
    license TEXT,
    tags TEXT, -- JSON array as text
    authors TEXT, -- JSON array as text
    dependencies TEXT, -- JSON object as text
    repository_url TEXT,
    documentation_url TEXT,
    homepage_url TEXT,
    issues_url TEXT,

    -- Version info
    requires_ansible TEXT,

    -- Upload metadata
    uploaded_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (collection_id) REFERENCES ansible_collections(id) ON DELETE CASCADE,
    FOREIGN KEY (artifact_sha256) REFERENCES artifacts(sha256) ON DELETE RESTRICT,
    UNIQUE (collection_id, version),
    CHECK (length(version) > 0)
);

CREATE INDEX idx_ansible_collection_versions_collection ON ansible_collection_versions(collection_id);
CREATE INDEX idx_ansible_collection_versions_version ON ansible_collection_versions(version);
CREATE INDEX idx_ansible_collection_versions_artifact ON ansible_collection_versions(artifact_sha256);
CREATE INDEX idx_ansible_collection_versions_created ON ansible_collection_versions(created_at DESC);

-- ============================================================================
-- Python Packages (PyPI)
-- ============================================================================

-- Python packages (e.g., "requests", "django")
CREATE TABLE IF NOT EXISTS python_packages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    normalized_name TEXT NOT NULL UNIQUE, -- PEP 503 normalized
    description TEXT,
    home_page TEXT,
    author TEXT,
    author_email TEXT,
    license TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (length(name) > 0),
    CHECK (length(normalized_name) > 0)
);

CREATE INDEX idx_python_packages_name ON python_packages(name);
CREATE INDEX idx_python_packages_normalized ON python_packages(normalized_name);

-- Python package versions (e.g., "requests" version "2.28.1")
CREATE TABLE IF NOT EXISTS python_package_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    package_id INTEGER NOT NULL,
    version TEXT NOT NULL,
    artifact_sha256 TEXT NOT NULL,

    -- Distribution info
    filename TEXT NOT NULL,
    python_version TEXT,
    requires_python TEXT,
    packagetype TEXT NOT NULL, -- "sdist", "bdist_wheel", etc.

    -- Metadata
    summary TEXT,
    keywords TEXT,
    classifiers TEXT, -- JSON array as text
    dependencies TEXT, -- JSON array as text

    -- Upload metadata
    uploaded_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (package_id) REFERENCES python_packages(id) ON DELETE CASCADE,
    FOREIGN KEY (artifact_sha256) REFERENCES artifacts(sha256) ON DELETE RESTRICT,
    UNIQUE (package_id, version, filename),
    CHECK (length(version) > 0),
    CHECK (length(filename) > 0),
    CHECK (packagetype IN ('sdist', 'bdist_wheel', 'bdist_egg'))
);

CREATE INDEX idx_python_package_versions_package ON python_package_versions(package_id);
CREATE INDEX idx_python_package_versions_version ON python_package_versions(version);
CREATE INDEX idx_python_package_versions_artifact ON python_package_versions(artifact_sha256);
CREATE INDEX idx_python_package_versions_created ON python_package_versions(created_at DESC);

-- ============================================================================
-- Container Images (OCI/Docker)
-- ============================================================================

-- Container repositories (e.g., "library/nginx", "myorg/myapp")
CREATE TABLE IF NOT EXISTS container_repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    is_public BOOLEAN NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (length(name) > 0)
);

CREATE INDEX idx_container_repositories_name ON container_repositories(name);
CREATE INDEX idx_container_repositories_public ON container_repositories(is_public);

-- Container tags (e.g., "nginx:latest", "nginx:1.21.0")
CREATE TABLE IF NOT EXISTS container_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    tag TEXT NOT NULL,
    manifest_sha256 TEXT NOT NULL,

    -- OCI manifest
    manifest_mediatype TEXT NOT NULL,
    manifest_size INTEGER NOT NULL,

    -- Config blob reference
    config_sha256 TEXT NOT NULL,
    config_mediatype TEXT,
    config_size INTEGER,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (repository_id) REFERENCES container_repositories(id) ON DELETE CASCADE,
    FOREIGN KEY (manifest_sha256) REFERENCES artifacts(sha256) ON DELETE RESTRICT,
    FOREIGN KEY (config_sha256) REFERENCES artifacts(sha256) ON DELETE RESTRICT,
    UNIQUE (repository_id, tag),
    CHECK (length(tag) > 0)
);

CREATE INDEX idx_container_tags_repository ON container_tags(repository_id);
CREATE INDEX idx_container_tags_tag ON container_tags(tag);
CREATE INDEX idx_container_tags_manifest ON container_tags(manifest_sha256);
CREATE INDEX idx_container_tags_updated ON container_tags(updated_at DESC);

-- Container layers (referenced by manifests)
CREATE TABLE IF NOT EXISTS container_layers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    manifest_sha256 TEXT NOT NULL,
    layer_sha256 TEXT NOT NULL,
    mediatype TEXT NOT NULL,
    size INTEGER NOT NULL,
    layer_index INTEGER NOT NULL,

    FOREIGN KEY (manifest_sha256) REFERENCES artifacts(sha256) ON DELETE CASCADE,
    FOREIGN KEY (layer_sha256) REFERENCES artifacts(sha256) ON DELETE RESTRICT,
    UNIQUE (manifest_sha256, layer_index),
    CHECK (size >= 0),
    CHECK (layer_index >= 0)
);

CREATE INDEX idx_container_layers_manifest ON container_layers(manifest_sha256);
CREATE INDEX idx_container_layers_layer ON container_layers(layer_sha256);

-- ============================================================================
-- Helm Charts
-- ============================================================================

-- Helm repositories
CREATE TABLE IF NOT EXISTS helm_repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (length(name) > 0)
);

CREATE INDEX idx_helm_repositories_name ON helm_repositories(name);

-- Helm charts
CREATE TABLE IF NOT EXISTS helm_charts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    artifact_sha256 TEXT NOT NULL,

    -- Chart metadata
    app_version TEXT,
    description TEXT,
    type TEXT, -- "application" or "library"
    keywords TEXT, -- JSON array as text
    home TEXT,
    sources TEXT, -- JSON array as text
    icon TEXT,

    -- Dependencies
    dependencies TEXT, -- JSON array as text

    -- Maintainers
    maintainers TEXT, -- JSON array as text

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (repository_id) REFERENCES helm_repositories(id) ON DELETE CASCADE,
    FOREIGN KEY (artifact_sha256) REFERENCES artifacts(sha256) ON DELETE RESTRICT,
    UNIQUE (repository_id, name, version),
    CHECK (length(name) > 0),
    CHECK (length(version) > 0)
);

CREATE INDEX idx_helm_charts_repository ON helm_charts(repository_id);
CREATE INDEX idx_helm_charts_name ON helm_charts(name);
CREATE INDEX idx_helm_charts_version ON helm_charts(version);
CREATE INDEX idx_helm_charts_artifact ON helm_charts(artifact_sha256);

-- ============================================================================
-- Generic File Storage (for RPM, DEB, etc.)
-- ============================================================================

-- Generic files with path-based access
CREATE TABLE IF NOT EXISTS generic_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    artifact_sha256 TEXT NOT NULL,

    -- Metadata
    filename TEXT NOT NULL,
    content_type TEXT,
    metadata TEXT, -- JSON object as text

    -- Upload info
    uploaded_by TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (artifact_sha256) REFERENCES artifacts(sha256) ON DELETE RESTRICT,
    CHECK (length(path) > 0),
    CHECK (length(filename) > 0)
);

CREATE INDEX idx_generic_files_path ON generic_files(path);
CREATE INDEX idx_generic_files_artifact ON generic_files(artifact_sha256);
CREATE INDEX idx_generic_files_filename ON generic_files(filename);

-- ============================================================================
-- Repository Access Control
-- ============================================================================

-- Repository users/tokens
CREATE TABLE IF NOT EXISTS repository_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    email TEXT,
    is_admin BOOLEAN NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (length(username) > 0)
);

CREATE INDEX idx_repository_users_username ON repository_users(username);

-- Access tokens for API authentication
CREATE TABLE IF NOT EXISTS access_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    scopes TEXT, -- JSON array as text
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES repository_users(id) ON DELETE CASCADE,
    CHECK (length(token_hash) > 0),
    CHECK (length(name) > 0)
);

CREATE INDEX idx_access_tokens_user ON access_tokens(user_id);
CREATE INDEX idx_access_tokens_token ON access_tokens(token_hash);
CREATE INDEX idx_access_tokens_expires ON access_tokens(expires_at);
