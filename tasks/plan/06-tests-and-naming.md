## 리팩토링: 테스트 디렉토리 네이밍/구조 정리

### 배경/목표
- 현재 `tests/`(integration/unit/contract/e2e 등)와 `testdata/`가 병존합니다.
- 커뮤니티에서는 상위 테스트 폴더를 `test/`로 쓰는 경우가 더 흔합니다. 팀 컨벤션에 맞춰 일관성을 확보합니다.

### 범위
- 선택: `tests/`를 `test/`로 리네이밍(또는 유지 시 명확한 사유를 문서화)
- 통합/계약/E2E/목업/헬퍼 하위 구조는 유지

### 단계별 작업 지침
1) 브랜치 생성
```bash
git checkout -b refactor/tests-naming
```

2) 디렉토리명 변경(선택)
```bash
git mv tests test 2>/dev/null || mv tests test
```

3) 문서/스크립트 경로 업데이트
```bash
grep -RIl "\btests/\b" . | xargs -I{} sed -i '' -e 's|tests/|test/|g' {}
```

4) 빌드/테스트
```bash
go test ./...
```

### 코드 영향 및 주의사항
- Go의 `*_test.go` 관례는 그대로 유지됩니다.
- CI가 `tests/` 경로를 하드코딩했다면 반드시 업데이트하세요.

### 검증 방법
- CI 파이프라인 실행 결과 확인
- 로컬에서 통합/E2E 스위트 실행

### 롤백 전략
```bash
git restore --staged -W .
git checkout -- .
git checkout -
git branch -D refactor/tests-naming
```

### 완료 기준 체크리스트
- [ ] `tests/` → `test/`로 일관화(또는 유지 사유 문서화)
- [ ] 관련 문서/스크립트 경로 업데이트
- [ ] 테스트 그린
