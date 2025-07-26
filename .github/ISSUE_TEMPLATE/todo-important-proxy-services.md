---
name: 🟠 중요 - 프록시 서비스 구현
about: Docker/NPM/APT 프록시 핵심 로직 구현
title: '[IMPORTANT] Implement proxy service logic'
labels: 'feature, important, todo'
assignees: ''
---

## 📍 위치
- **Docker**: `internal/services/proxy/docker_service.go:43`
- **NPM**: `internal/services/proxy/npm_service.go:43`
- **APT**: `internal/services/proxy/apt_service.go:44`

## 🔍 현재 상황
각 패키지 매니저별 프록시 서비스의 핵심 로직이 구현되지 않았습니다.

## ✅ 구현 필요 사항

### Docker Service
- [ ] Docker Registry API v2 프록시 구현
- [ ] 이미지 레이어 캐싱
- [ ] 매니페스트 처리
- [ ] 인증 토큰 중계

### NPM Service
- [ ] NPM Registry API 프록시 구현
- [ ] 패키지 메타데이터 캐싱
- [ ] Tarball 다운로드 처리
- [ ] 검색 기능 구현

### APT Service
- [ ] APT Repository 프록시 구현
- [ ] Packages 파일 파싱 및 캐싱
- [ ] deb 패키지 다운로드
- [ ] GPG 서명 검증

## 🚨 우선순위
**중요** - 핵심 기능 구현

## 📚 참고사항
- 각 서비스의 어댑터 패턴 활용
- 기존 캐시 시스템과 통합
- 에러 처리 및 재시도 로직 포함
