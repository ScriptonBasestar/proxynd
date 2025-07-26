---
name: 🟡 중간 - 캐시 시스템 개선
about: 캐시 관련 기능 개선 및 구현
title: '[MEDIUM] Improve cache system'
labels: 'enhancement, performance, todo'
assignees: ''
---

## 📍 위치
- `cache/eviction.go:153`
- `internal/plugins/adapters/npm_adapter.go:282`
- `internal/services/docker/cache_manager.go:414`

## 🔍 현재 상황
캐시 시스템의 여러 부분에서 개선이 필요한 TODO가 있습니다.

## ✅ 구현 필요 사항

### cache/eviction.go:153
- [ ] 디렉토리 스캔으로 캐시 키 목록 가져오기
- [ ] 파일 시스템 기반 키 수집
- [ ] 성능 최적화

### npm_adapter.go:282
- [ ] NPM 패키지 캐시 조회 로직
- [ ] 캐시 히트/미스 처리
- [ ] 메타데이터 캐싱

### cache_manager.go:414
- [ ] 더 정교한 캐시 제거 정책
- [ ] LFU (Least Frequently Used) 구현
- [ ] 적응형 캐시 크기 조절
- [ ] 캐시 통계 기반 최적화

## 🚨 우선순위
**중간** - 성능 개선 사항

## 📚 참고사항
- 기존 LRU 정책과 호환성 유지
- 메모리 효율성 고려
- 동시성 안전성 보장
