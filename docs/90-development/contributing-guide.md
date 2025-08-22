# Contributing to ProxyND

우리는 모든 기여를 환영합니다! ProxyND는 GitLab 모델을 따라 **모든 코드가 오픈소스**입니다.

## 🤝 기여 방법

### 1. 개발 환경 설정
```bash
# 저장소 포크 및 클론
git clone https://github.com/yourusername/proxynd.git
cd proxynd

# 개발 환경 설정
make dev-prepare
make dev-setup
```

### 2. 브랜치 전략
- `main`: 안정된 릴리스
- `develop`: 개발 브랜치
- `feature/*`: 새 기능
- `fix/*`: 버그 수정

### 3. 커밋 컨벤션
```
type(scope): subject

body

footer
```

**타입**:
- `feat`: 새 기능
- `fix`: 버그 수정
- `docs`: 문서 개선
- `test`: 테스트 추가
- `refactor`: 리팩토링
- `chore`: 빌드, 도구 등

**예시**:
```
feat(cache): add S3 backend support

- Implement S3 storage adapter
- Add configuration options
- Include unit tests

Closes #123
```

## 🏢 엔터프라이즈 기능 기여

**중요**: 엔터프라이즈 기능도 오픈소스입니다!

### 엔터프라이즈 기능 개발 가이드
1. 모든 엔터프라이즈 기능은 `internal/enterprise/` 디렉토리에 위치
2. 기능 플래그로 제어되어야 함
3. 라이센스 없이도 코드는 빌드되어야 함
4. 테스트에서는 모의 라이센스 사용

### 예시: 새 엔터프라이즈 기능 추가
```go
// internal/enterprise/features/newfeature.go
package features

import "github.com/yourusername/proxynd/internal/enterprise/license"

func NewEnterpriseFeature(gate *license.FeatureGate) *EnterpriseFeature {
    return &EnterpriseFeature{
        enabled: gate.IsEnabled("new_feature"),
    }
}

func (f *EnterpriseFeature) Execute() error {
    if !f.enabled {
        return ErrFeatureNotLicensed
    }
    // 실제 구현
}
```

## 🧪 테스트

### 테스트 실행
```bash
# 단위 테스트
make test

# 커버리지 포함
make test-coverage

# 통합 테스트
make test-integration
```

### 테스트 작성 가이드
- 모든 새 기능은 테스트 필수
- 엔터프라이즈 기능은 라이센스 있/없는 경우 모두 테스트
- 최소 80% 커버리지 목표

## 📚 문서화

### 문서 위치
- 사용자 문서: `docs/`
- API 문서: 코드 주석 (godoc)
- 예제: `examples/`

### 문서 작성 가이드
1. 한국어/영어 모두 환영
2. 명확한 예시 포함
3. 엔터프라이즈 기능은 명시적 표시

## 🐛 이슈 보고

### 버그 보고
```markdown
**설명**: 버그에 대한 명확한 설명

**재현 방법**:
1. 첫 번째 단계
2. 두 번째 단계
3. ...

**예상 동작**: 어떻게 동작해야 하는지

**실제 동작**: 실제로 어떻게 동작하는지

**환경**:
- ProxyND 버전:
- OS:
- Go 버전:
```

### 기능 요청
- 사용 사례 설명
- 제안하는 해결책
- 대안 고려사항

## 🔍 코드 리뷰

### 리뷰어 체크리스트
- [ ] 코드 스타일 준수
- [ ] 테스트 포함
- [ ] 문서 업데이트
- [ ] 성능 영향 검토
- [ ] 보안 고려사항
- [ ] 라이센스 호환성

### 자동 검사
```bash
# 린트
make lint

# 보안 스캔
make security

# 품질 검사
make quality
```

## 📜 라이센스

기여하신 모든 코드는 AGPL-3.0 라이센스가 적용됩니다.
엔터프라이즈 기능 포함 모든 코드가 공개됩니다.

## 🤖 AI 지원 개발 (AI-Assisted Development)

### AI 협업 가드레일
ProxyND는 **Hexagonal + Clean Architecture** 기반의 대규모 리팩토링을 진행 중입니다. AI를 활용한 개발 시 반드시 준수해야 할 가이드라인:

📋 **필수 문서**: [CLAUDE.md](./CLAUDE.md) - AI 협업 가드레일

### 핵심 원칙
```bash
# 의존성 방향 (엄격 준수)
adapters → ports → usecase → domain

# 금지된 의존성
❌ domain → usecase/ports/adapters
❌ usecase → adapters  
❌ ports → adapters
```

### 대규모 리팩토링 프로토콜
1. **Phase 1**: 파일 이동만 (별도 커밋)
2. **Phase 2**: import 경로 수정 (별도 커밋)  
3. **Phase 3**: 아키텍처 개선 (별도 커밋)

### 보호 구역 (수정 금지)
- `scripts/verify-api-endpoints.sh` (회귀 테스트 기준)
- `Makefile`, `Makefile.*.mk` (빌드 시스템)
- `docker-compose.e2e.yml` (E2E 환경)
- `README.md` 개발 워크플로 섹션

### 검증 체크리스트
```bash
make build              # 빌드 성공 필수
make test-unit          # 단위 테스트 통과
make verify-api         # API 회귀 검증 (CRITICAL)
```

## 🏛️ 아키텍처 가이드

### 계층별 책임
| 계층 | 위치 | 책임 | 의존성 |
|------|------|------|--------|
| Domain | `internal/domain/` | 비즈니스 규칙, 엔티티 | 없음 |
| Usecase | `internal/usecase/` | 애플리케이션 로직 | Domain, Ports |
| Ports | `internal/ports/` | 인터페이스 정의 | Domain |
| Adapters | `internal/adapters/` | 외부 구현체 | Ports, Usecase, Domain |

### 파일 이동 가이드
```bash
# 현재 → 목표
handlers/           → internal/adapters/http/fiber/handlers/
routers/           → internal/adapters/http/fiber/routers/
middlewares/       → internal/adapters/http/fiber/middleware/
cache/             → internal/adapters/cache/
internal/services/ → internal/usecase/
```

## 🙋 도움 요청

- Discord: [참여하기](https://discord.gg/proxynd)
- GitHub Discussions: [토론 참여](https://github.com/yourusername/proxynd/discussions)
- Email: contribute@proxynd.io

## 🎯 좋은 첫 기여

`good first issue` 라벨이 붙은 이슈를 확인하세요!

## Commit Guidelines

### Commit Message Format
`{prefix}({AI툴}): {요약}`
- `prefix`: `feat`, `fix`, `refactor`, `test`, or `chore`
- `AI툴`: `claude`, `gemini`, `cursor`, `roocode`, or `none`
- `요약`: 50 characters max

### Using Commit Helper
```bash
# Interactive mode:
./scripts/commit_helper.sh

# Manual commit:
./scripts/commit_helper.sh commit <prefix> <ai_tool> "<summary>"
./scripts/commit_helper.sh commit feat claude "결제 연동 모듈 추가"

# Partial staging (for multi-category commits):
./scripts/commit_helper.sh stage
git reset -p  # Unstage specific hunks
git add -p    # Stage specific hunks
./scripts/commit_helper.sh commit <prefix> <ai_tool> "<summary>"
```

### Workflow
1. Make code changes (AI-assisted or manual)
2. Run `./scripts/commit_helper.sh` (interactive) or use staged commands
3. Repeat for each logical change set
