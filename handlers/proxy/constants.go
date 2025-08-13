package proxy

// MIME types
const (
	MimeApplicationGzip                 = "application/gzip"
	MimeApplicationXGzip                = "application/x-gzip"
	MimeTextPlain                       = "text/plain"
	MimeApplicationPGPSignature         = "application/pgp-signature"
	MimeApplicationOctetStream          = "application/octet-stream"
	MimeApplicationZip                  = "application/zip"
	MimeApplicationJSON                 = "application/json"
	MimeApplicationJavaArchive          = "application/java-archive"
	MimeApplicationXML                  = "application/xml"
	MimeApplicationDebianBinaryPackage  = "application/vnd.debian.binary-package"
	MimeApplicationXBzip2               = "application/x-bzip2"
	MimeApplicationDockerManifestV2JSON = "application/vnd.docker.distribution.manifest.v2+json"
	MimeApplicationJSONCharsetUTF8      = "application/json; charset=utf-8"
	MimeTextPlainCharsetUTF8            = "text/plain; charset=utf-8"
	MimeApplicationXGzipContentType     = "application/x-gzip"
)

// Proxy types
const (
	ProxyTypeAPT    = "apt"
	ProxyTypeMaven  = "maven"
	ProxyTypeDocker = "docker"
	ProxyTypeNPM    = "npm"
	ProxyTypePIP    = "pip"
	ProxyTypeYUM    = "yum"
	ProxyTypeAPK    = "apk"
)

// Common strings
const (
	DockerAPIV2Path        = "v2/"
	FormatJSON             = "json"
	TypeDirectory          = "directory"
	TypeFile               = "file"
	LevelGroup             = "group"
	LevelArtifact          = "artifact"
	LevelVersion           = "version"
	DockerTagsEndpoint     = "tags"
	DockerListEndpoint     = "list"
	StatusHealthy          = "healthy"
	StatusUnhealthy        = "unhealthy"
	BackendRedis           = "redis"
	BackendFile            = "file"
	FieldTypeError         = "error"
	FieldTypeDuration      = "duration"
	AuthTypeBasic          = "basic"
	AuthTypeBearer         = "bearer"
	DockerResourceManifest = "manifest"
	DockerResourceBlob     = "blob"
	HeaderHost             = "Host"
	ResultSuccess          = "success"
	ResultFailure          = "failure"
	MethodGet              = "GET"
	MethodPOST             = "POST"
	PriorityMedium         = "medium"
	YumMetadataPrimary     = "primary"
	ExtGz                  = ".gz"
)
