package logging

// Common field names used in logging
const (
	fieldError     = "error"
	fieldPath      = "path"
	fieldFile      = "file"
	fieldUserID    = "user_id"
	fieldRequestID = "request_id"
	fieldStatus    = "status"
)

// Output formats
const (
	formatJSON = "json"
	formatText = "text"
)

// Output destinations
const (
	outputStdout = "stdout"
	outputBoth   = "both"
)

// Header names
const (
	headerRequestID = "X-Request-ID"
)
