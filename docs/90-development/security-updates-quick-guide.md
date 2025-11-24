# 🔒 보안 업데이트 빠른 가이드

**목적**: GitHub Security 탭의 보안 취약점을 자동으로 처리

---

## ⚡ 빠른 시작 (30초)

### 로컬에서 즉시 실행

```bash
# 1. 보안 업데이트 확인
cd proxynd-core
./scripts/merge-security-updates.sh --dry-run

# 2. 보안 업데이트 병합
./scripts/merge-security-updates.sh
```

**끝!** 보안 PR이 자동으로 rebase merge됩니다.

---

## 🤖 GitHub Actions 자동화 (한 번만 설정)

### 1단계: 워크플로우 활성화

```bash
git add .github/workflows/security-auto-merge.yml
git commit -m "ci(security): add security auto-merge workflow"
git push origin master
```

### 2단계: GitHub 설정

**Repository Settings** → **Actions** → **General**:
- ✅ Workflow permissions: **Read and write permissions**
- ✅ **Allow GitHub Actions to create and approve pull requests**

### 완료!

이제 Dependabot이 보안 PR을 생성하면 자동으로:
1. 승인됨
2. CI 체크 대기
3. 통과 시 rebase merge

---

## 📊 방법 비교

| 방법 | 시간 | 자동화 | 설정 필요 |
|------|------|--------|----------|
| **로컬 스크립트** | 30초 | ❌ 수동 | ✅ 없음 |
| **GitHub Actions** | 5분 | ✅ 자동 | ⚠️ 한 번만 |

---

## 🔍 보안 업데이트 확인

### GitHub 웹에서

1. Repository → **Security** 탭
2. **Dependabot alerts** 섹션
3. 취약점 목록 확인

### 터미널에서

```bash
# 보안 PR 목록 확인
gh pr list --repo ScriptonBasestar/proxynd --label security

# 또는 스크립트 사용
./scripts/merge-security-updates.sh --dry-run
```

---

## 🚨 긴급 보안 업데이트 처리

### 즉시 병합 (CI 무시)

```bash
# PR 번호 확인
gh pr list --repo ScriptonBasestar/proxynd --label security

# 즉시 승인 및 병합
gh pr review 123 --approve --repo ScriptonBasestar/proxynd
gh pr merge 123 --rebase --repo ScriptonBasestar/proxynd
```

**주의**: CI 체크를 건너뛰므로 긴급 상황에만 사용하세요.

---

## 📋 체크리스트

### 첫 설정 (한 번만)

- [ ] GitHub Actions 워크플로우 커밋
- [ ] Repository Actions 권한 설정
- [ ] 스크립트 실행 권한 확인 (`chmod +x scripts/*.sh`)

### 정기 확인 (주 1회)

- [ ] `./scripts/merge-security-updates.sh --dry-run` 실행
- [ ] 보안 PR 확인
- [ ] 필요시 `./scripts/merge-security-updates.sh` 실행

### 보안 알림 발생 시

- [ ] GitHub Security 탭 확인
- [ ] Dependabot PR 생성 확인
- [ ] GitHub Actions 자동 병합 대기 또는
- [ ] 로컬 스크립트로 수동 병합

---

## 🆘 문제 해결

### "No mergeable security updates found"

**원인**: 보안 PR이 없거나 병합 불가 상태

**확인**:
```bash
# 모든 Dependabot PR 확인
gh pr list --repo ScriptonBasestar/proxynd --author "app/dependabot"
```

### GitHub Actions가 병합하지 않음

**체크**:
1. Workflow permissions 설정 확인
2. Branch protection rules 확인
3. CI 체크 통과 여부 확인

**로그 확인**:
```
Repository → Actions → Security Updates Auto-Merge 워크플로우
```

### 스크립트 실행 오류

```bash
# 권한 확인
ls -l scripts/merge-security-updates.sh

# 권한 부여
chmod +x scripts/merge-security-updates.sh

# gh CLI 인증 확인
gh auth status
```

---

## 💡 팁

### 1. 보안 업데이트만 선택적으로

```bash
./scripts/merge-security-updates.sh
```

### 2. 모든 업데이트 한번에

```bash
./scripts/merge-security-updates.sh --all
```

### 3. Dry-run으로 미리 확인

```bash
./scripts/merge-security-updates.sh --dry-run
```

### 4. GitHub 웹에서 확인

```
https://github.com/ScriptonBasestar/proxynd/security/dependabot
```

---

## 🔗 관련 문서

- **상세 가이드**: [`dependabot-auto-merge-setup.md`](./dependabot-auto-merge-setup.md)
- **일반 업데이트**: [`scripts/merge-dependabot-updates.sh`](../../scripts/merge-dependabot-updates.sh)
- **Dependabot 설정**: [`.github/dependabot.yml`](../../.github/dependabot.yml)

---

**마지막 업데이트**: 2025-11-24
