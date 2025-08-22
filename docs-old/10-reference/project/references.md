
# Package Manager Proxy Reference

## Maven
* https://cwiki.apache.org/confluence/display/MAVENOLD/Repository+Layout+-+Final
* https://cwiki.apache.org/confluence/display/MAVENOLD/Repository+Metadata

### Maven 패키지 흐름
1. **요청 구조**: `GET /maven/{groupId}/{artifactId}/{version}/{filename}`
2. **캐시 경로**: `storage/proxy/maven/{groupId}/{artifactId}/{version}/{filename}`
3. **업스트림 요청**: `{upstream_url}/{groupId}/{artifactId}/{version}/{filename}`
4. **메타데이터**: maven-metadata.xml, pom.xml 파일 처리

### Maven 주요 헤더
- `Content-Type`: application/java-archive, application/xml
- `Last-Modified`: 파일 수정 시간
- `ETag`: 파일 버전 식별자
- `Cache-Control`: public, max-age=3600

## APT
* https://wiki.debian.org/DebianRepository/Format
* http://www.ibiblio.org/gferg/ldp/giles/repository/repository-2.html

### APT 패키지 흐름
1. **요청 구조**: `GET /apt/{osType}/{path}`
   - osType: ubuntu, debian 등
   - path: dists/focal/main/binary-amd64/Packages 등
2. **캐시 경로**: `storage/proxy/apt/{path}` (osType 제외)
3. **업스트림 선택**: osType에 따라 적절한 미러 서버 선택
4. **특수 파일**: Release, Packages, Sources 파일 처리

### APT 주요 헤더
- `Content-Type`: text/plain, application/x-debian-package
- `Content-Encoding`: gzip (Packages.gz, Sources.gz)
- `Last-Modified`: 레포지토리 업데이트 시간
- `Cache-Control`: public, max-age=300 (메타데이터는 짧은 TTL)

### APT 요청 예시
```
GET /apt/ubuntu/dists/focal/Release
GET /apt/ubuntu/dists/focal/main/binary-amd64/Packages.gz
GET /apt/ubuntu/pool/main/a/apache2/apache2_2.4.41-4ubuntu3_amd64.deb
```

## NPM
### NPM 패키지 흐름
1. **요청 구조**:
   - 패키지 메타데이터: `GET /npm/{package_name}`
   - 패키지 다운로드: `GET /npm/{package_name}/-/{package_name}-{version}.tgz`
2. **캐시 경로**: `storage/proxy/npm/{package_name}/...`
3. **업스트림**: npmjs.org, yarn, taobao 미러 중 선택
4. **스코프 패키지**: `@scope/package` 형태 지원

### NPM 주요 헤더
- `Content-Type`: application/json, application/octet-stream
- `Accept`: application/vnd.npm.install-v1+json
- `npm-auth-token`: 인증 토큰 (선택사항)
- `Cache-Control`: public, max-age=300

### NPM 요청 예시
```
GET /npm/express
GET /npm/express/-/express-4.18.2.tgz
GET /npm/@types/node
GET /npm/@types/node/-/node-18.15.0.tgz
```

## PIP (PyPI)
### PIP 패키지 흐름
1. **요청 구조**:
   - 패키지 인덱스: `GET /pip/simple/{package_name}/`
   - 패키지 다운로드: `GET /pip/packages/{hash}/{filename}`
2. **캐시 경로**: `storage/proxy/pip/simple/{package_name}/...`
3. **업스트림**: pypi.org, douban, aliyun 미러 중 선택
4. **API 버전**: Simple API v1 지원

### PIP 주요 헤더
- `Content-Type`: text/html, application/octet-stream
- `Accept`: text/html, application/vnd.pypi.simple.v1+html
- `User-Agent`: pip/xx.x.x
- `Cache-Control`: public, max-age=600

### PIP 요청 예시
```
GET /pip/simple/requests/
GET /pip/packages/a5/61/a867851fd5ab77277495a8709ddda0861b28d7f4c4c6a31e1dcf25c44/requests-2.28.2-py3-none-any.whl
```

## Docker Registry
### Docker 패키지 흐름
1. **요청 구조**:
   - 매니페스트: `GET /v2/{name}/manifests/{reference}`
   - 블롭: `GET /v2/{name}/blobs/{digest}`
2. **캐시 경로**: `storage/proxy/docker/v2/{name}/...`
3. **업스트림**: docker.io, gcr.io, quay.io 중 선택
4. **API 버전**: Registry API v2 지원

### Docker 주요 헤더
- `Content-Type`: application/vnd.docker.distribution.manifest.v2+json
- `Docker-Content-Digest`: SHA256 다이제스트
- `Authorization`: Bearer 토큰
- `Accept`: application/vnd.docker.distribution.manifest.v2+json

## Go Proxy
* https://github.com/elazarl/goproxy

### Go 모듈 흐름
1. **요청 구조**:
   - 모듈 정보: `GET /{module}/@v/list`
   - 모듈 다운로드: `GET /{module}/@v/{version}.zip`
2. **캐시 경로**: `storage/proxy/go/{module}/@v/...`
3. **업스트림**: proxy.golang.org, goproxy.cn 등
4. **GOPROXY 프로토콜**: Go 1.13+ 표준 지원

### Go Proxy 주요 헤더
- `Content-Type`: text/plain, application/zip
- `Cache-Control`: public, immutable, max-age=31536000
- `Disable-Module-Fetch`: true (모듈 자동 가져오기 비활성화)

## 캐시 정책 및 TTL
- **메타데이터**: 5-30분 (빈번한 업데이트)
- **패키지 파일**: 1-24시간 (변경 빈도 낮음)
- **바이너리**: 무제한 (불변 파일)
- **Release/Snapshot**: 구분하여 다른 TTL 적용
