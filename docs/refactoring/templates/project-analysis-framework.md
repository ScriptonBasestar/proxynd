# 프로젝트 분석 프레임워크

## 개요
이 프레임워크는 리팩토링 전 프로젝트를 체계적으로 분석하기 위한 범용 가이드입니다.

## 🔍 프로젝트 분석 체크리스트

### 1. 프로젝트 개요 파악

#### 기본 정보 수집
- [ ] **프로젝트 타입 확인**
  - [ ] Web Application (API, Full-stack)
  - [ ] Library/Framework
  - [ ] CLI Tool
  - [ ] Desktop Application
  - [ ] Mobile Application
  - [ ] Microservice
  - [ ] Monolith
  - [ ] 기타: ___________

- [ ] **기술 스택 조사**
  - [ ] 주 프로그래밍 언어 및 버전
  - [ ] 프레임워크 및 주요 라이브러리
  - [ ] 빌드 도구 및 패키지 매니저
  - [ ] 테스트 프레임워크
  - [ ] 배포 환경

- [ ] **프로젝트 규모 측정**
  ```bash
  # 예시 명령어 (언어별로 조정 필요)
  find . -name "*.{확장자}" | wc -l  # 파일 수
  cloc .  # 코드 라인 수 (cloc 도구 사용)
  ```

### 2. 코드베이스 구조 분석

#### 디렉토리 구조 매핑
```
프로젝트_루트/
├── {소스_코드_디렉토리}/
│   ├── {기능별_모듈}/
│   ├── {공통_컴포넌트}/
│   └── {유틸리티}/
├── {테스트_디렉토리}/
├── {설정_파일들}/
├── {문서}/
└── {빌드_관련}/
```

#### 주요 컴포넌트 식별
- [ ] **진입점 (Entry Points)**
  - 메인 함수/클래스 위치
  - 초기화 과정
  - 라우팅/디스패칭 메커니즘

- [ ] **핵심 비즈니스 로직**
  - 도메인 모델 위치
  - 비즈니스 규칙 구현부
  - 핵심 알고리즘

- [ ] **인프라 레이어**
  - 데이터 접근 계층
  - 외부 서비스 통합
  - 설정 관리

### 3. 코드 품질 지표 수집

#### 정적 분석 도구 실행
```bash
# 언어별 정적 분석 도구 예시
# Go: golangci-lint, staticcheck
# Python: pylint, flake8, mypy
# JavaScript: ESLint, JSHint
# Java: SpotBugs, PMD, Checkstyle
```

#### 측정할 메트릭
- [ ] **코드 복잡도**
  - 순환 복잡도 (Cyclomatic Complexity)
  - 인지 복잡도 (Cognitive Complexity)
  - 중첩 깊이 (Nesting Depth)

- [ ] **코드 중복**
  - 중복 코드 블록 수
  - 중복 비율 (%)
  - 주요 중복 패턴

- [ ] **의존성 분석**
  - 순환 의존성
  - 결합도 (Coupling)
  - 응집도 (Cohesion)

- [ ] **테스트 커버리지**
  - 라인 커버리지
  - 브랜치 커버리지
  - 함수 커버리지

### 4. 문제 영역 식별

#### 코드 스멜 탐지
- [ ] **구조적 문제**
  - [ ] God Class/Module (너무 큰 클래스/모듈)
  - [ ] Feature Envy (다른 클래스의 데이터 과다 사용)
  - [ ] Data Clumps (데이터 덩어리)
  - [ ] Primitive Obsession (원시 타입 과다 사용)

- [ ] **중복 패턴**
  - [ ] 복사-붙여넣기 코드
  - [ ] 유사한 알고리즘 반복
  - [ ] 보일러플레이트 코드

- [ ] **복잡성 문제**
  - [ ] 긴 메서드/함수
  - [ ] 긴 매개변수 목록
  - [ ] 복잡한 조건문
  - [ ] 깊은 중첩

#### 아키텍처 문제
- [ ] **레이어 위반**
  - 프레젠테이션 → 데이터 직접 접근
  - 순환 의존성
  - 불명확한 책임 경계

- [ ] **확장성 제약**
  - 하드코딩된 값
  - 단일 구현에 대한 강한 결합
  - 테스트 어려움

### 5. 성능 분석

#### 성능 병목 지점 파악
- [ ] **프로파일링 실행**
  ```bash
  # 언어별 프로파일링 도구
  # Go: pprof
  # Python: cProfile, py-spy
  # Java: JProfiler, YourKit
  # Node.js: clinic.js, 0x
  ```

- [ ] **주요 확인 사항**
  - [ ] 느린 함수/메서드
  - [ ] 메모리 누수
  - [ ] 비효율적인 알고리즘
  - [ ] 과도한 I/O 작업
  - [ ] 불필요한 객체 생성

### 6. 보안 취약점 스캔

#### 보안 도구 실행
```bash
# 보안 스캔 도구 예시
# 일반: SonarQube, Snyk
# Go: gosec
# Python: bandit, safety
# JavaScript: npm audit, ESLint security plugins
# Java: OWASP Dependency Check
```

#### 확인할 취약점
- [ ] **코드 레벨**
  - [ ] SQL Injection
  - [ ] XSS 취약점
  - [ ] 하드코딩된 비밀정보
  - [ ] 안전하지 않은 암호화

- [ ] **의존성**
  - [ ] 알려진 취약점이 있는 라이브러리
  - [ ] 오래된 버전 사용
  - [ ] 라이선스 문제

## 📊 분석 결과 정리 템플릿

### 1. 핵심 지표 요약
```markdown
## 프로젝트 현황 대시보드

### 규모
- 총 파일 수: {number}
- 총 코드 라인: {number}
- 주요 모듈 수: {number}

### 품질 지표
- 평균 복잡도: {number}
- 코드 중복률: {percent}%
- 테스트 커버리지: {percent}%
- 기술 부채 추정: {time/cost}

### 우선순위 문제
1. {가장 심각한 문제}
2. {두 번째 문제}
3. {세 번째 문제}
```

### 2. 상세 분석 보고서 구조
```markdown
## 상세 분석 결과

### 아키텍처 분석
- **현재 구조**: {설명}
- **주요 문제점**: {리스트}
- **개선 기회**: {리스트}

### 코드 품질 분석
- **복잡도 분포**: {차트/테이블}
- **중복 코드 위치**: {파일 리스트}
- **리팩토링 대상**: {우선순위 리스트}

### 성능 분석
- **병목 지점**: {리스트}
- **최적화 기회**: {리스트}
- **예상 개선 효과**: {메트릭}

### 보안 분석
- **발견된 취약점**: {리스트}
- **위험도 평가**: {High/Medium/Low}
- **권장 조치**: {액션 아이템}
```

## 🛠️ 분석 도구 모음

### 범용 도구
- **코드 분석**: SonarQube, CodeClimate
- **의존성 분석**: Dependency Cruiser, Structure101
- **시각화**: CodeScene, Gource

### 언어별 특화 도구

#### Go
- 정적 분석: `golangci-lint`, `staticcheck`
- 테스트: `go test -cover`
- 벤치마크: `go test -bench`

#### Python
- 정적 분석: `pylint`, `flake8`, `mypy`
- 복잡도: `radon`, `mccabe`
- 테스트: `pytest --cov`

#### JavaScript/TypeScript
- 정적 분석: `ESLint`, `TSLint`
- 번들 분석: `webpack-bundle-analyzer`
- 테스트: `jest --coverage`

#### Java
- 정적 분석: `SpotBugs`, `PMD`
- 복잡도: `JaCoCo`, `Cobertura`
- 아키텍처: `ArchUnit`

## 📋 AI 프롬프트용 분석 요약 템플릿

```markdown
# 프로젝트 분석 요약

## 기본 정보
- 프로젝트: {name}
- 타입: {type}
- 언어: {language}
- 규모: {size}

## 주요 발견사항
1. **가장 큰 문제**: {description}
   - 영향 범위: {scope}
   - 심각도: {severity}
   
2. **두 번째 문제**: {description}
   - 영향 범위: {scope}
   - 심각도: {severity}

## 리팩토링 우선순위
1. {highest_priority_area}
2. {second_priority_area}
3. {third_priority_area}

## 제약사항
- {constraint_1}
- {constraint_2}
- {constraint_3}
```

이 템플릿을 사용하여 AI에게 구체적인 리팩토링 계획을 요청할 수 있습니다.