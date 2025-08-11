package proxy

// MIME types
const (
	mimeApplicationGzip                 = "application/gzip"
	mimeApplicationXGzip                = "application/x-gzip"
	mimeTextPlain                       = "text/plain"
	mimeApplicationPGPSignature         = "application/pgp-signature"
	mimeApplicationOctetStream          = "application/octet-stream"
	mimeApplicationZip                  = "application/zip"
	mimeApplicationJSON                 = "application/json"
	mimeApplicationJavaArchive          = "application/java-archive"
	mimeApplicationXML                  = "application/xml"
	mimeApplicationDebianBinaryPackage  = "application/vnd.debian.binary-package"
	mimeApplicationXBzip2               = "application/x-bzip2"
	mimeApplicationDockerManifestV2JSON = "application/vnd.docker.distribution.manifest.v2+json"
	mimeApplicationJSONCharsetUTF8      = "application/json; charset=utf-8"
	mimeTextPlainCharsetUTF8            = "text/plain; charset=utf-8"
)

// Proxy types
const (
	proxyTypeAPT    = "apt"
	proxyTypeMaven  = "maven"
	proxyTypeDocker = "docker"
	proxyTypeNPM    = "npm"
	proxyTypePIP    = "pip"
	proxyTypeYUM    = "yum"
	proxyTypeAPK    = "apk"
)

// Common strings
const (
	dockerAPIV2Path        = "v2/"
	formatJSON             = "json"
	typeDirectory          = "directory"
	typeFile               = "file"
	levelGroup             = "group"
	levelArtifact          = "artifact"
	levelVersion           = "version"
	dockerTagsEndpoint     = "tags"
	dockerListEndpoint     = "list"
	statusHealthy          = "healthy"
	statusUnhealthy        = "unhealthy"
	backendRedis           = "redis"
	backendFile            = "file"
	fieldTypeError         = "error"
	fieldTypeDuration      = "duration"
	authTypeBasic          = "basic"
	authTypeBearer         = "bearer"
	dockerResourceManifest = "manifest"
	dockerResourceBlob     = "blob"
)
