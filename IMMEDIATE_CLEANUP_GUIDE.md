# 🧹 ProxyND 즉시 정리 가이드

## 📋 정리 대상 파일 목록

### 1. 로그 파일 (루트 디렉토리)
```bash
# 인증 관련 로그
auth_debug_detailed.log
auth_final_test.log
auth_success_test.log
auth_test.log
auth_working_test.log
debug_auth_verbose.log
debug_auth.log
final_auth_test.log

# 서버 로그
server_debug.log
server_fixed.log
server_improved.log
server_new.log
server.log
```

### 2. 테스트 아티팩트
```bash
configs.test
middlewares.test
proxy_test.test
coverage.html
coverage.out
```

### 3. 임시 스크립트
```bash
automated_lint_fixes.sh
temp_delete.sh
fix_defer_close.sh
fix_lint_issues.sh
run_lint.sh
run_mod_tidy.sh
test_lint.sh
delete_file.sh
move_file.py
```

### 4. 분석 보고서
```bash
LINT_ANALYSIS_COMPLETE.md
lint_analysis.md
LINT_FIX_SUMMARY.md
lint_fixes_summary.md
complex_comments.txt
todo_items.json
```

## 🚀 즉시 실행 명령어

### Step 1: 백업 디렉토리 생성
```bash
# 백업 디렉토리 생성
BACKUP_DIR="backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"
echo "백업 디렉토리 생성됨: $BACKUP_DIR"
```

### Step 2: 로그 파일 백업 및 제거
```bash
# 로그 파일 백업
mv *.log "$BACKUP_DIR/" 2>/dev/null || echo "로그 파일 없음"

# Git에서 추적 중인 로그 파일 제거
git rm --cached *.log 2>/dev/null || true
```

### Step 3: 테스트 아티팩트 제거
```bash
# 테스트 관련 파일 제거
rm -f *.test coverage.out coverage.html
```

### Step 4: 임시 스크립트 정리
```bash
# legacy 디렉토리 생성 및 스크립트 이동
mkdir -p scripts/legacy
mv automated_lint_fixes.sh temp_delete.sh fix_defer_close.sh \
   fix_lint_issues.sh run_lint.sh run_mod_tidy.sh test_lint.sh \
   delete_file.sh move_file.py scripts/legacy/ 2>/dev/null || true
```

### Step 5: 분석 보고서 정리
```bash
# 분석 보고서 디렉토리 생성 및 파일 이동
mkdir -p docs/analysis_reports
mv LINT_*.md lint_*.md complex_comments.txt todo_items.json \
   docs/analysis_reports/ 2>/dev/null || true
```

### Step 6: .gitignore 업데이트
```bash
# .gitignore에 패턴 추가
cat >> .gitignore << 'EOL'

# === ProxyND Cleanup ===
# Logs
*.log
logs/
*_log.txt

# Test artifacts
*.test
coverage.out
coverage.html
*.cover
.coverage_history

# Temporary files
*.tmp
*.temp
*.bak
*.swp
*.swo
*~

# Build artifacts
proxynd
proxyndctl
dist/
build/

# Local environment
.env.local
.env.*.local

# Backup directories
backup_*/
EOL
```

### Step 7: Go 모듈 정리
```bash
# Go 모듈 정리
go mod tidy
go mod download
```

### Step 8: Git 정리
```bash
# Git 상태 확인
git status

# 변경사항 스테이징
git add .

# 커밋
git commit -m "chore: 프로젝트 정리 - 임시 파일 및 로그 제거"
```

## 📊 정리 전후 비교

### Before
```
proxynd/
├── auth_debug_detailed.log    # 로그 파일들
├── server_*.log               # 서버 로그들
├── *.test                     # 테스트 아티팩트
├── automated_lint_fixes.sh    # 임시 스크립트
├── LINT_*.md                  # 분석 보고서
└── ...
```

### After
```
proxynd/
├── backup_20250124_*/         # 백업된 파일들
├── scripts/legacy/            # 임시 스크립트 보관
├── docs/analysis_reports/     # 분석 보고서 보관
└── .gitignore                 # 업데이트됨
```

## ⚠️ 주의사항

1. **백업 확인**: 삭제하기 전에 반드시 백업 디렉토리를 확인하세요.
2. **Git 상태**: `git status`로 추적되는 파일을 확인하세요.
3. **팀 공유**: 정리 작업 전 팀원들에게 알리세요.

## 🔄 정기 정리 작업

### 일일 정리
```bash
# 로그 파일 정리
find . -name "*.log" -mtime +7 -delete
```

### 주간 정리
```bash
# 테스트 아티팩트 정리
make clean

# Docker 정리
docker system prune -f
```

### 월간 정리
```bash
# 오래된 백업 제거
find . -name "backup_*" -mtime +30 -exec rm -rf {} \;

# Git 정리
git gc --aggressive --prune=now
```

## 📝 체크리스트

- [ ] 백업 디렉토리 생성 완료
- [ ] 로그 파일 백업 완료
- [ ] 테스트 아티팩트 제거 완료
- [ ] 임시 스크립트 이동 완료
- [ ] 분석 보고서 정리 완료
- [ ] .gitignore 업데이트 완료
- [ ] Go 모듈 정리 완료
- [ ] Git 커밋 완료
- [ ] 팀 공유 완료

---

**작성일**: 2025-01-24  
**다음 정리**: 2025-01-31
