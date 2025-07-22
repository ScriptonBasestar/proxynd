# Dependabot 브랜치 관리 가이드

## 📋 개요

ProxyND 프로젝트는 Dependabot을 사용하여 의존성 업데이트를 자동화합니다. 하지만 Dependabot 브랜치들이 누적되면서 리포지토리를 어지럽힐 수 있어 체계적인 관리가 필요합니다.

## 🔧 자동 정리 시스템

### GitHub Actions 자동 정리
- **일정**: 매주 일요일 오전 2시 (UTC+9)
- **조건**: 
  - 메인 브랜치에 병합된 브랜치
  - 30일 이상 오래된 브랜치
- **워크플로우**: `.github/workflows/cleanup.yml`

### 수동 정리 스크립트
```bash
# 즉시 정리 실행
./scripts/cleanup-dependabot-branches.sh

# 또는 GitHub Actions에서 수동 실행
# Actions > Cleanup and Maintenance > Run workflow
```

## 📊 Dependabot 설정 최적화

### 그룹화 전략
의존성 업데이트를 논리적으로 그룹화하여 PR 수를 줄입니다:

1. **aws-sdk**: AWS SDK 관련 모든 업데이트
2. **networking**: HTTP/네트워킹 라이브러리
3. **dev-tools**: 테스트 및 개발 도구
4. **cli-tools**: 터미널 및 CLI 도구
5. **security-updates**: 보안 업데이트 (최우선)
6. **github-actions**: GitHub Actions 업데이트

### PR 제한
- **Go 모듈**: 최대 5개 PR
- **GitHub Actions**: 최대 3개 PR
- **Docker**: 최대 5개 PR

## 🎯 브랜치 명명 규칙

Dependabot이 생성하는 브랜치 이름 패턴:
```
dependabot-{ecosystem}-{group-name}-{hash}
dependabot-{ecosystem}-{package-name}-{version}
```

예시:
- `dependabot-go_modules-aws-sdk-1a2b3c4d`
- `dependabot-github_actions-github-actions-a1b2c3d`
- `dependabot-go_modules-minor-updates-11d3a79bdf`

## 🧹 수동 정리 명령어

### 현재 Dependabot 브랜치 확인
```bash
git branch -r | grep dependabot
```

### 개별 브랜치 삭제
```bash
# 원격 브랜치 삭제
git push origin --delete dependabot-브랜치명

# 로컬 추적 브랜치 정리
git remote prune origin
```

### 대량 정리 (신중하게!)
```bash
# 모든 dependabot 브랜치 나열
git branch -r | grep dependabot | sed 's/origin\///' > dependabot_branches.txt

# 확인 후 일괄 삭제 (예시)
cat dependabot_branches.txt | xargs -I {} git push origin --delete {}
```

## ⚠️ 주의사항

### 삭제하면 안 되는 브랜치
- 현재 진행 중인 PR의 브랜치
- 최근 생성된 브랜치 (7일 이내)
- 보안 업데이트 브랜치

### 삭제해도 되는 브랜치
- 이미 병합된 브랜치
- 30일 이상 오래된 브랜치
- 닫힌 PR의 브랜치

## 📈 모니터링

### 정기 확인사항
1. **주간 리뷰**: 매주 월요일 생성된 PR 검토
2. **월간 정리**: 매월 첫째 주 브랜치 상태 점검
3. **분기별 감사**: 분기마다 Dependabot 설정 최적화

### 알림 설정
- PR 생성 시 자동 알림
- 보안 업데이트 즉시 알림
- 월간 정리 리포트

## 🔄 워크플로우 예시

### 일반적인 처리 과정
1. **월요일 오전**: Dependabot이 새 PR 생성
2. **화요일**: 개발팀이 PR 검토 및 병합
3. **일요일 새벽**: 자동 정리 스크립트 실행
4. **분기별**: 설정 최적화 검토

### 응급 상황 처리
```bash
# 브랜치가 너무 많을 때 (50개 이상)
./scripts/cleanup-dependabot-branches.sh

# GitHub UI에서 확인
# Settings > Branches에서 브랜치 보호 규칙 확인
```

## 💡 팁과 트릭

### 효율적인 PR 관리
- 보안 업데이트는 즉시 병합
- 마이너 업데이트는 그룹으로 검토
- 메이저 업데이트는 신중히 테스트

### 브랜치 이름으로 빠른 파악
```bash
# AWS 관련 브랜치만 보기
git branch -r | grep dependabot | grep aws

# 보안 업데이트만 보기
git branch -r | grep dependabot | grep security
```

### 자동화 개선
- 테스트 통과 시 자동 병합 고려
- 특정 패키지는 자동 승인 설정
- 충돌 발생 시 자동 알림

## 📚 관련 문서

- [Dependabot 공식 문서](https://docs.github.com/en/code-security/dependabot)
- [ProxyND CI/CD 가이드](../docs/CI_CD.md)
- [브랜치 관리 정책](../docs/BRANCHING.md)

---

## 🆘 문제 해결

### 자주 발생하는 문제

#### Q: 브랜치가 자동으로 삭제되지 않아요
A: `cleanup.yml` 워크플로우가 활성화되어 있는지 확인하고, 수동으로 스크립트를 실행해보세요.

#### Q: PR이 너무 많이 생성되어요
A: `dependabot.yml`에서 `open-pull-requests-limit`을 줄이고 그룹화를 강화하세요.

#### Q: 중요한 업데이트를 놓쳤어요
A: 보안 업데이트는 별도 그룹으로 분리되어 있으니 Security 탭을 확인하세요.

---

*이 문서는 ProxyND 프로젝트의 Dependabot 브랜치 관리를 위한 가이드입니다.*