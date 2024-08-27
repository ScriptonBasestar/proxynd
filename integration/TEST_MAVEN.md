


```bash
curl -O https://repo1.maven.org/maven2/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar
curl -O https://repo1.maven.org/maven2/com/google/guava/guava/31.1-jre/guava-31.1-jre.jar
curl -O https://repo1.maven.org/maven2/org/slf4j/slf4j-api/1.7.36/slf4j-api-1.7.36.jar
curl -O https://repo1.maven.org/maven2/junit/junit/4.13.2/junit-4.13.2.jar
curl -O https://repo1.maven.org/maven2/org/springframework/spring-core/5.3.27/spring-core-5.3.27.jar

export REPO_URL=http://localhost:8080/proxy/maven
curl -O $REPO_URL/org/apache/commons/commons-lang3/3.12.0/commons-lang3-3.12.0.jar
curl -O $REPO_URL/com/google/guava/guava/31.1-jre/guava-31.1-jre.jar
curl -O $REPO_URL/org/slf4j/slf4j-api/1.7.36/slf4j-api-1.7.36.jar
curl -O $REPO_URL/junit/junit/4.13.2/junit-4.13.2.jar
curl -O $REPO_URL/org/springframework/spring-core/5.3.27/spring-core-5.3.27.jar
```

cachedir에 파일 확인
