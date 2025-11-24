# Dependabot Auto-Merge Setup

**목적**: Dependabot PR을 자동으로 rebase merge하여 linear 히스토리 유지

---

## 📋 목차

1. [방법 비교](#방법-비교)
2. [🔒 보안 업데이트 자동화 (우선)](#보안-업데이트-자동화-우선)
3. [방법 1: GitHub Actions (자동)](#방법-1-github-actions-자동)
4. [방법 2: 로컬 스크립트 (수동)](#방법-2-로컬-스크립트-수동)
5. [GitHub Repository 설정](#github-repository-설정)

---

## 방법 비교

| 방법 | 자동화 | 설정 복잡도 | 권장 상황 |
|------|--------|-------------|----------|
| **🔒 보안 업데이트 자동화** | ✅ 완전 자동 | ✅ 간단 | **모든 환경 (최우선)** |
| **GitHub Actions** | ✅ 완전 자동 | ⚠️ 중간 (branch protection 필요) | 프로덕션 환경 |
| **로컬 스크립트** | ❌ 수동 실행 | ✅ 간단 | 개발/테스트 환경 |

---

## 🔒 보안 업데이트 자동화 (우선)

**보안 취약점은 최우선으로 자동 처리됩니다.**

### GitHub Actions: 보안 업데이트 전용 워크플로우

**파일**: `.github/workflows/security-auto-merge.yml`

**특징**:
- ✅ 보안 라벨(`security`) 또는 키워드(`CVE`, `vulnerability`) 감지
- ✅ 즉시 자동 승인 및 rebase merge
- ✅ 일반 업데이트는 수동 리뷰 필요
- ✅ CI 체크 통과 후 병합
- ✅ 실패 시 코멘트로 알림

**활성화**:
```bash
git add .github/workflows/security-auto-merge.yml
git commit -m "ci(security): add security auto-merge workflow"
git push origin master
```

**작동 방식**:
1. Dependabot이 보안 PR 생성
2. GitHub Actions가 `security` 라벨 감지
3. 즉시 승인 및 auto-merge 활성화
4. CI 통과 후 자동으로 rebase merge

### 로컬 스크립트: 보안 업데이트만 병합

**파일**: `scripts/merge-security-updates.sh`

**사용법**:
```bash
# 보안 업데이트만 확인
./scripts/merge-security-updates.sh --dry-run

# 보안 업데이트만 병합
./scripts/merge-security-updates.sh

# 모든 Dependabot PR 병합 (보안 포함)
./scripts/merge-security-updates.sh --all
```

**특징**:
- 🔒 보안 라벨 자동 감지
- 🔍 CVE, vulnerability 키워드 검색
- ⚡ 보안 PR 우선 표시
- ✅ 즉시 실행 가능 (설정 불필요)

**출력 예시**:
```
🔒 Found 2 SECURITY update(s):

  🔒 #123: chore(deps): bump golang.org/x/net (CVE-2024-1234)
     Labels: security, dependencies, go

  #124: chore(deps): bump minor-updates group
     Labels: dependencies
```

### GitHub Security 탭 연동

GitHub Security 탭(`/security`)의 Dependabot alerts는 자동으로:

1. **Dependabot이 보안 PR 생성** → `security` 라벨 자동 추가
2. **GitHub Actions 워크플로우 트리거** → 즉시 승인 및 병합 대기
3. **CI 통과** → 자동 rebase merge
4. **보안 알림 자동 해결**

**수동 확인이 필요한 경우**:
- Major 버전 업데이트
- 여러 패키지 동시 업데이트
- Breaking changes 포함

---

## 방법 1: GitHub Actions (자동)

### 워크플로우 파일

2개의 워크플로우가 제공됩니다:

1. **`.github/workflows/dependabot-auto-merge.yml`** (고급)
   - CI 체크 대기
   - Major 버전 업데이트는 수동 리뷰 필요
   - Minor/Patch만 자동 병합

2. **`.github/workflows/dependabot-auto-merge-simple.yml`** (간단)
   - 모든 Dependabot PR 자동 승인 및 병합
   - 빠른 처리

### 활성화 방법

1. **워크플로우 파일 커밋**:
   ```bash
   git add .github/workflows/dependabot-auto-merge*.yml
   git commit -m "ci(deps): add dependabot auto-merge workflows"
   git push origin master
   ```

2. **GitHub Repository 설정** (아래 섹션 참조)

---

## 방법 2: 로컬 스크립트 (수동)

### 스크립트 위치

`scripts/merge-dependabot-updates.sh`

### 사용법

```bash
# 기본 사용 (rebase merge)
./scripts/merge-dependabot-updates.sh

# Dry-run (실행 전 확인)
./scripts/merge-dependabot-updates.sh --dry-run

# Squash merge (단일 커밋으로 병합)
./scripts/merge-dependabot-updates.sh --squash

# Rebase merge (linear 히스토리, 기본값)
./scripts/merge-dependabot-updates.sh --rebase

# Standard merge (merge commit 생성)
./scripts/merge-dependabot-updates.sh --merge
```

### 스크립트 동작

1. **PR 검색**: Mergeable 상태인 Dependabot PR 찾기
2. **확인 요청**: 사용자에게 병합 여부 확인 (dry-run 제외)
3. **자동 승인**: 각 PR 승인
4. **Rebase 병합**: Linear 히스토리 유지하며 병합
5. **결과 요약**: 성공/실패 개수 출력

### 장점

- ✅ Branch protection 설정 불필요
- ✅ 즉시 사용 가능
- ✅ Merge 방식 선택 가능
- ✅ Dry-run 지원

### 단점

- ❌ 수동 실행 필요
- ❌ PR 생성 시 자동 처리 안 됨

---

## GitHub Repository 설정

GitHub Actions 자동화를 위해서는 다음 설정이 필요합니다.

### 1. Branch Protection Rules 설정

**Settings** → **Branches** → **Add rule** (master 브랜치):

```
✅ Require a pull request before merging
   ✅ Require approvals (1)
   ✅ Dismiss stale pull request approvals when new commits are pushed

✅ Require status checks to pass before merging
   ✅ Require branches to be up to date before merging
   Status checks:
   - build
   - test
   - lint (선택)

✅ Allow auto-merge

⚠️ Do not require pull request reviews for administrators
   (GitHub Actions가 승인할 수 있도록)
```

### 2. Actions Permissions 설정

**Settings** → **Actions** → **General**:

```
✅ Allow all actions and reusable workflows

Workflow permissions:
✅ Read and write permissions
✅ Allow GitHub Actions to create and approve pull requests
```

### 3. Auto-merge 활성화

Repository settings:

```
Settings → General → Pull Requests
✅ Allow auto-merge
```

### 4. GITHUB_TOKEN Permissions

워크플로우 파일에 이미 포함됨:

```yaml
permissions:
  contents: write
  pull-requests: write
```

---

## 테스트 방법

### 1. 로컬 스크립트 테스트

```bash
# Dry-run으로 확인
./scripts/merge-dependabot-updates.sh --dry-run

# 실제 병합
./scripts/merge-dependabot-updates.sh --rebase
```

### 2. GitHub Actions 테스트

1. Dependabot PR 생성될 때까지 대기
2. **Actions** 탭에서 워크플로우 실행 확인
3. PR이 자동으로 승인 및 병합되는지 확인

---

## 문제 해결

### "Protected branch rules not configured"

**원인**: Branch protection 미설정

**해결**:
- Repository Settings → Branches에서 branch protection 설정
- "Allow auto-merge" 활성화

### "No such tool available"

**원인**: GitHub CLI 미설치

**해결**:
```bash
# macOS
brew install gh

# Linux
sudo apt install gh

# 인증
gh auth login
```

### 워크플로우가 실행되지 않음

**원인**: Workflow permissions 부족

**해결**:
- Settings → Actions → General
- "Read and write permissions" 선택
- "Allow GitHub Actions to create and approve pull requests" 활성화

---

## 권장 워크플로우

### 개발 단계

```bash
# 수동으로 확인 후 병합
./scripts/merge-dependabot-updates.sh --dry-run
./scripts/merge-dependabot-updates.sh --rebase
```

### 프로덕션 단계

1. **Branch protection 설정** 완료
2. **GitHub Actions 활성화**
3. **자동 병합** 작동

---

## 추가 참고

### Rebase vs Squash vs Merge

| 방식 | 히스토리 | 커밋 개수 | 사용 시기 |
|------|---------|----------|----------|
| **Rebase** | Linear | 원본 유지 | 각 커밋이 의미 있을 때 |
| **Squash** | Linear | 1개로 합침 | Dependabot PR (권장) |
| **Merge** | Non-linear | 원본 + Merge commit | Feature 브랜치 |

### Dependabot 설정

`.github/dependabot.yml`에서 그룹화 설정:

```yaml
groups:
  minor-updates:
    update-types:
      - "minor"
      - "patch"
    patterns:
      - "*"
```

이렇게 하면 여러 의존성을 하나의 PR로 묶어서 병합 횟수를 줄일 수 있습니다.

---

**마지막 업데이트**: 2025-11-24
