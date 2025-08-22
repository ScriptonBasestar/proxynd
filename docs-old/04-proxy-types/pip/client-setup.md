# PIP 프록시 설정 가이드

## 클라이언트 설정

### 1. pip 설정 파일 수정

#### ~/.pip/pip.conf (Linux/Mac) 또는 %APPDATA%\pip\pip.ini (Windows)

```ini
[global]
index-url = http://your-proxy-server:8080/proxy/pip/simple
trusted-host = your-proxy-server
```

### 2. 환경 변수 설정

```bash
# 임시 설정
export PIP_INDEX_URL=http://your-proxy-server:8080/proxy/pip/simple
export PIP_TRUSTED_HOST=your-proxy-server

# 영구 설정 (~/.bashrc 또는 ~/.zshrc에 추가)
echo 'export PIP_INDEX_URL=http://your-proxy-server:8080/proxy/pip/simple' >> ~/.bashrc
echo 'export PIP_TRUSTED_HOST=your-proxy-server' >> ~/.bashrc
```

### 3. 명령줄 옵션 사용

```bash
# 단일 명령에서 프록시 사용
pip install --index-url http://your-proxy-server:8080/proxy/pip/simple --trusted-host your-proxy-server requests

# 특정 패키지 설치
pip install --index-url http://your-proxy-server:8080/proxy/pip/simple numpy pandas
```

## 서버 설정 (ProxyND)

### pip-proxy.yaml 설정 예시

```yaml
path: proxy/pip
use_cache: true

proxies:
  - name: pypi
    url: https://pypi.org
  - name: douban
    url: https://pypi.doubanio.com
  - name: aliyun
    url: https://mirrors.aliyun.com/pypi
```

## 가상 환경에서 사용

### venv
```bash
# 가상 환경 생성
python -m venv myenv

# 활성화
source myenv/bin/activate  # Linux/Mac
# 또는
myenv\Scripts\activate  # Windows

# pip 설정
pip config set global.index-url http://your-proxy-server:8080/proxy/pip/simple
pip config set global.trusted-host your-proxy-server
```

### conda
```bash
# conda 환경에서 pip 사용 시
conda activate myenv
pip config set global.index-url http://your-proxy-server:8080/proxy/pip/simple
```

## 캐시 동작

- 첫 번째 요청 시 PyPI에서 패키지를 다운로드하고 로컬에 캐시
- 이후 요청은 캐시된 파일을 제공
- 패키지 메타데이터는 TTL에 따라 갱신

## 문제 해결

### SSL 인증서 오류
```bash
# SSL 검증 비활성화 (권장하지 않음)
pip install --index-url http://your-proxy-server:8080/proxy/pip/simple \
            --trusted-host your-proxy-server \
            --cert /dev/null \
            package-name
```

### 캐시 클리어
```bash
# 클라이언트 캐시 클리어
pip cache purge

# 서버 캐시 클리어 (ProxyND 서버에서)
rm -rf /storage/proxy/pip/*
```

### 디버그 모드
```bash
# 상세 로그 확인
pip install -v --index-url http://your-proxy-server:8080/proxy/pip/simple package-name
```

## 프라이빗 패키지

프라이빗 패키지 저장소를 사용하는 경우:

```yaml
# pip-proxy.yaml에 추가
proxies:
  - name: private
    url: https://your-private-pypi.com
    # 인증이 필요한 경우 (향후 지원 예정)
    # auth:
    #   username: user
    #   password: pass
```
