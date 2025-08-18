## 리팩토링: 런타임 산출물/운영 자산 재배치

### 배경/목표
- 런타임 산출물(`logs/`, `tmp/`, `cache/`)과 운영/배포 자산(`monitoring/`, `systemd/`, `helm/`)이 루트에 혼재되어 있습니다.
- 목표: 코드와 분리하고 표준 경로로 재배치하여 가독성과 유지보수성을 높입니다.

### 범위
- 런타임 산출물
  - `logs/`, `tmp/`, `cache/` → VCS 제외 유지 또는 `var/` 하위로 이동
- 운영/배포 자산
  - `monitoring/` → `deployments/monitoring/`
  - `systemd/` → `deployments/systemd/` 또는 `init/systemd/`
  - `helm/` → `deployments/helm/`
- 문서/스크립트의 경로 갱신 포함

### 단계별 작업 지침
1) 브랜치 생성
```bash
git checkout -b refactor/ops-assets-reorg
```

2) 디렉토리 생성 및 이동
```bash
# 런타임 산출물은 git 관리 제외를 권장(.gitignore 확인)
mkdir -p var/logs var/tmp var/cache deployments/monitoring deployments/systemd deployments/helm

# 운영/배포 자산 이동
git mv monitoring/* deployments/monitoring/ 2>/dev/null || mv monitoring/* deployments/monitoring/

git mv systemd/* deployments/systemd/ 2>/dev/null || mv systemd/* deployments/systemd/

git mv helm/* deployments/helm/ 2>/dev/null || mv helm/* deployments/helm/
```

3) 참조 경로 업데이트
```bash
grep -RIl "\bmonitoring/\b" . | xargs -I{} sed -i '' -e 's|monitoring/|deployments/monitoring/|g' {}

grep -RIl "\bsystemd/\b" . | xargs -I{} sed -i '' -e 's|systemd/|deployments/systemd/|g' {}

grep -RIl "\bhelm/\b" . | xargs -I{} sed -i '' -e 's|helm/|deployments/helm/|g' {}
```

4) .gitignore 정리(필요 시)
```bash
echo "logs/" >> .gitignore
echo "tmp/" >> .gitignore
echo "cache/" >> .gitignore
echo "var/" >> .gitignore
```

### 코드 영향 및 주의사항
- 코드가 런타임 디렉토리를 절대경로로 가정하지 않았는지 확인하세요. 환경변수/설정으로 경로를 주입하는 방식을 권장합니다.
- CI/CD 스크립트와 배포 문서에서 경로 변경 사항을 반영하세요.

### 검증 방법
- 로컬/테스트 환경에서 애플리케이션을 실행해 로그/캐시/임시파일 생성 위치를 확인
- Helm/monitoring/systemd 관련 스크립트/문서가 새 경로에서 정상 동작하는지 확인

### 롤백 전략
```bash
git restore --staged -W .
git checkout -- .
git checkout -
git branch -D refactor/ops-assets-reorg
```

### 완료 기준 체크리스트
- [ ] 운영/배포 자산이 `deployments/` 하위로 이동
- [ ] 런타임 산출물이 VCS에서 제외되거나 `var/` 하위로 이동
- [ ] 모든 문서/스크립트의 경로가 업데이트됨
