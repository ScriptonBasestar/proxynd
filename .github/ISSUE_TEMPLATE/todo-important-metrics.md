---
name: 🟠 중요 - 메트릭 시스템 완성
about: Prometheus 메트릭 연동 및 실제 값 추출
title: '[IMPORTANT] Complete metrics system implementation'
labels: 'monitoring, important, todo'
assignees: ''
---

## 📍 위치
- **파일**: `routers/metrics_router.go`
- **TODO 개수**: 5개

## 🔍 현재 상황
메트릭 라우터에 여러 미구현 부분이 있어 모니터링 시스템이 불완전합니다.

## ✅ 구현 필요 사항

### Line 71
- [ ] 설정에서 사용자 정보 로드
```go
// TODO: 설정에서 사용자 정보 로드
```

### Line 239, 263
- [ ] 실제 메트릭에서 값 추출
- [ ] Prometheus 메트릭에서 값 추출
```go
// TODO: 실제 메트릭에서 값 추출
// TODO: 실제 Prometheus 메트릭에서 값 추출
```

### Line 270
- [ ] 메트릭 수집 실제 구현
```go
// TODO: 실제 구현
```

### Line 276
- [ ] 건강 상태 확인 로직
```go
// TODO: 실제 건강 상태 확인 로직
```

## 🚨 우선순위
**중요** - 모니터링 및 운영에 필수적인 기능

## 📚 참고사항
- Prometheus 클라이언트 라이브러리 활용
- 기존 메트릭 컬렉터와 통합
- Grafana 대시보드 호환성 고려
