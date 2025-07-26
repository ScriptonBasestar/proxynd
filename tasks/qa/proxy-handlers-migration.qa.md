# ✅ 프록시 핸들러 마이그레이션 QA 시나리오

## related_tasks
- /tasks/done/phase-2/004-apt-handler-migration__DONE_20250717.md
- /tasks/done/phase-2/005-maven-handler-migration__DONE_20250717.md

## purpose
APT와 Maven 프록시 핸들러가 의존성 주입 패턴으로 올바르게 마이그레이션되었는지 확인

## scenario

### 1. APT 프록시 기능 테스트
1. ProxyND 서버 시작
   ```bash
   make dev-run
   ```

2. APT 프록시 엔드포인트 테스트
   ```bash
   # Release 파일 요청 (첫 번째 - MISS)
   curl -v http://localhost:8080/proxy/apt/dists/focal/Release
   
   # 응답 헤더 확인
   # X-Cache-Status: MISS 확인
   ```

3. 동일 요청 재실행 (캐시 HIT 확인)
   ```bash
   # Release 파일 요청 (두 번째 - HIT)
   curl -v http://localhost:8080/proxy/apt/dists/focal/Release
   
   # 응답 헤더 확인
   # X-Cache-Status: HIT 확인
   ```

4. 패키지 파일 다운로드 테스트
   ```bash
   curl -v http://localhost:8080/proxy/apt/pool/main/v/vim/vim_8.2.deb -o test.deb
   
   # 파일 크기 및 무결성 확인
   file test.deb
   ```

### 2. Maven 프록시 기능 테스트
1. Maven POM 파일 요청
   ```bash
   # Spring Core POM 요청 (첫 번째 - MISS)
   curl -v http://localhost:8080/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.pom
   
   # X-Cache-Status: MISS 확인
   ```

2. Maven JAR 파일 다운로드
   ```bash
   curl -v http://localhost:8080/proxy/maven/org/springframework/spring-core/5.3.21/spring-core-5.3.21.jar \
        -o spring-core.jar
   
   # JAR 파일 검증
   jar tf spring-core.jar | head -10
   ```

3. Maven 메타데이터 요청
   ```bash
   curl -v http://localhost:8080/proxy/maven/org/springframework/spring-core/maven-metadata.xml
   ```

### 3. 성능 및 메모리 테스트
1. 동시 요청 처리 확인
   ```bash
   # 50개 동시 연결로 1000개 요청
   ab -n 1000 -c 50 http://localhost:8080/proxy/apt/dists/focal/Release
   ```

2. 메모리 사용량 모니터링
   ```bash
   # 서버 시작 전후 메모리 사용량 비교
   ps aux | grep proxynd
   ```

### 4. 설정 변경 테스트
1. APT 프록시 비활성화
   - `apt-proxy.yaml`에서 `enabled: false` 설정
   - 서버 재시작 없이 설정 리로드
   - APT 엔드포인트 요청 → 404 응답 확인

2. 미러 서버 변경
   - Maven 설정에서 repository URL 변경
   - 새로운 저장소에서 패키지 다운로드 확인

## expected_result

### ✅ 기능 검증
- [ ] APT/Maven 프록시 요청이 정상적으로 처리됨
- [ ] 캐시 HIT/MISS가 올바르게 동작함
- [ ] 파일 다운로드가 정상적으로 완료됨
- [ ] 에러 발생 시 적절한 에러 응답 반환

### ✅ 성능 검증
- [ ] 동시 요청 처리 시 에러율 1% 미만
- [ ] 평균 응답 시간 2초 이내
- [ ] 메모리 누수 없음
- [ ] CPU 사용률 안정적

### ✅ 설정 검증
- [ ] 설정 변경이 즉시 반영됨
- [ ] 프록시 비활성화 시 404 응답
- [ ] 미러 서버 변경이 정상 동작

## tags
[qa, e2e, manual, grouped, proxy, migration]