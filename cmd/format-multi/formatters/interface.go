package formatters

// Formatter 는 각 언어별 포맷터가 구현해야 하는 인터페이스
type Formatter interface {
	// Name 은 포맷터의 이름을 반환 (예: "gofumpt")
	Name() string

	// Language 는 언어 이름을 반환 (예: "Go")
	Language() string

	// IsAvailable 은 포맷터가 시스템에 설치되어 있는지 확인
	IsAvailable() bool

	// Install 은 포맷터를 설치
	Install() error

	// Format 은 파일을 포맷팅
	Format(filename string, config interface{}) error
}
