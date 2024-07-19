# 모은다 

## 용도

Proxy, Mirror를 같이 하려고 했는데... 귀찮아서 proxy만 남길 예정

- [x] Maven
- [x] Apt
- [ ] Npm
- [ ] Go
- [ ] Python
- [ ] Ruby
- [ ] Docker
- [ ] 쩌 Repo 인증기능
- [ ] 이 리포 인증기능 

## 쓰면 좋은점
로컬 repo 충돌났을 때 다 날리고 다시 하는 경우 좋음
컴퓨터가 두서너대여섯대인 경우 좋음
인터넷이 느린 경우 좋음
사무실에서 다 같이 쓰는경우 좋음

## 쓰면 나쁨점
한번하면 되지만 설치는 귀찮음
지원되는게 별로 없음

## 사용법

대강 이런걸 어느 경로에 저장할까 고민으 좀 해 보는 중

- global config
  - 베이스 설정 루프
  - 베이스 캐시 패스
  - 캐시 시간
- PackageManager config
  - repo별로 cache 설정
- Registry config
  - 각각 target cache 설정
```yaml
path: d:/tmp/cachedir/proxy/maven
# https://docs.gradle.org/current/userguide/declaring_repositories.html
servers:
  - name: Maven Center
    id: maven-center
    url: https://repo.maven.apache.org/maven2/
    description: "maven centeral"
    
  - name: jcener
    url: https://jcenter.bintray.com
    description: "jcenter"
#  google:
#    url: https://maven.google.com
#    description: "google android"
#  # https://repo.spring.io/webapp/#/home
#  spring_release:
#    name: repo.spring.io-releases
#    url: https://repo.spring.io/release
#  spring_snapshorts:
#    id: spring-snapshots
#    name: repo.spring.io-snapshots
#    url: https://repo.spring.io/snapshot

#  company_internal:
#    url: http://company
#    description: "my company"
#    auth:
#      method: BASIC
#      username:
```

## REF

maven
* https://cwiki.apache.org/confluence/display/MAVENOLD/Repository+Layout+-+Final
* https://cwiki.apache.org/confluence/display/MAVENOLD/Repository+Metadata
apt
* https://wiki.debian.org/DebianRepository/Format
* http://www.ibiblio.org/gferg/ldp/giles/repository/repository-2.html

goproxy
* https://github.com/elazarl/goproxy
