# 🔧 AI 기반 리팩토링 프롬프트 시스템

## 개요

이 디렉토리는 AI를 활용한 체계적인 코드 리팩토링을 위한 범용 프롬프트와 가이드를 제공합니다. 어떤 프로젝트에서도 활용할 수 있도록 설계되었으며, 프로젝트 타입과 언어에 관계없이 적용 가능합니다.

## 🎯 목적

- **체계적 접근**: 단계별 리팩토링 프로세스 제공
- **AI 최적화**: AI 컨텍스트 한계를 고려한 효율적 구조
- **범용성**: 다양한 프로젝트 타입과 언어 지원
- **실용성**: 즉시 적용 가능한 템플릿과 가이드

## 📁 디렉토리 구조

```
docs/refactoring/
├── README.md                           # 이 파일
├── USAGE_GUIDE.md                      # 상세 사용 가이드
├── templates/                          # 범용 템플릿
│   ├── universal-prompt-template.md    # 리팩토링 프롬프트 템플릿
│   ├── project-analysis-framework.md   # 프로젝트 분석 프레임워크
│   ├── universal-principles.md         # 범용 리팩토링 원칙
│   ├── phase-template.md              # Phase별 프롬프트 템플릿
│   └── project-types/                 # 프로젝트 타입별 템플릿
│       ├── web-api/
│       ├── library/
│       ├── cli-tool/
│       └── microservice/
├── phases/                             # 리팩토링 단계별 문서
│   └── plan.md                        # 마스터 플랜 템플릿
└── examples/                          # 실제 적용 예시
    └── proxynd/                       # ProxyND 프로젝트 예시
```

## 🚀 빠른 시작

### 1. 프로젝트 분석
```bash
# 프로젝트 기본 정보 수집
cloc .  # 코드 규모 파악

# project-analysis-framework.md 참고하여 분석 수행
```

### 2. 리팩토링 계획 수립
```bash
# phases/plan.md를 복사하여 프로젝트에 맞게 수정
cp phases/plan.md my-refactoring-plan.md
```

### 3. AI 프롬프트 활용
```markdown
# universal-prompt-template.md를 참고하여 AI에게 요청
"이 템플릿을 사용하여 [프로젝트명]의 [리팩토링 영역]에 대한 
상세 계획을 작성해주세요."
```

## 📋 핵심 구성요소

### 1. [프로젝트 분석 프레임워크](templates/project-analysis-framework.md)
- 체계적인 코드베이스 분석 체크리스트
- 문제점 식별 가이드
- 우선순위 결정 프레임워크

### 2. [범용 리팩토링 원칙](templates/universal-principles.md)
- 언어/프레임워크 중립적 원칙
- 안전한 리팩토링 프로세스
- 품질 지표 및 목표

### 3. [마스터 플랜](phases/plan.md)
- 6단계 리팩토링 로드맵
- AI 컨텍스트 최적화 전략
- 검증 체크리스트

### 4. [사용 가이드](USAGE_GUIDE.md)
- 단계별 실행 방법
- 프로젝트 타입별 가이드
- 실용적인 팁과 함정 회피

## 🎯 지원하는 프로젝트 타입

- **Web API**: RESTful API, GraphQL, gRPC
- **Library/Framework**: 재사용 가능한 컴포넌트
- **CLI Tool**: 명령줄 도구
- **Microservice**: 분산 시스템 컴포넌트
- **Monolith**: 대규모 단일 애플리케이션
- **Mobile App**: iOS/Android 애플리케이션
- **Desktop App**: 데스크톱 애플리케이션

## 💡 주요 특징

### AI 컨텍스트 최적화
- Phase당 ~100K 토큰 이하로 제한
- 단계별 집중 분석
- 효율적인 프롬프트 구조

### 점진적 접근
- Phase 0: 즉시 실행 가능한 정리
- Phase 1-4: 핵심 리팩토링
- Phase 5-6: 장기 개선 사항

### 검증 가능한 결과
- 정량적 지표 측정
- 단계별 체크포인트
- 롤백 가능한 변경

## 📊 예상 효과

- **코드 품질**: 30-50% 개선
- **개발 생산성**: 40-60% 향상
- **유지보수 비용**: 50% 이상 절감
- **버그 발생률**: 60-70% 감소

## 🛠️ 필수 도구

### 범용 도구
- Git (버전 관리)
- 정적 분석 도구 (SonarQube, CodeClimate)
- 코드 포맷터 (언어별)

### 언어별 도구
- **Go**: golangci-lint, gofmt
- **Python**: black, pylint, mypy
- **JavaScript**: ESLint, Prettier
- **Java**: SpotBugs, Checkstyle

## 🤝 기여 방법

1. 새로운 프로젝트 타입 템플릿 추가
2. 실제 적용 사례 공유
3. 도구 및 팁 추가
4. 번역 및 현지화

## 📚 참고 자료

- [Refactoring by Martin Fowler](https://martinfowler.com/books/refactoring.html)
- [Clean Code by Robert C. Martin](https://www.oreilly.com/library/view/clean-code-a/9780136083238/)
- [Working Effectively with Legacy Code](https://www.oreilly.com/library/view/working-effectively-with/0131177052/)

## 🏷️ 버전

- **현재 버전**: v1.0.0
- **최종 업데이트**: 2025-01-01
- **호환성**: 모든 프로젝트 타입 및 언어

## 📝 라이선스

이 프로젝트는 MIT 라이선스 하에 공개됩니다. 자유롭게 사용, 수정, 배포할 수 있습니다.

---

**시작하기**: [USAGE_GUIDE.md](USAGE_GUIDE.md)를 참고하여 첫 리팩토링을 시작하세요! 🚀