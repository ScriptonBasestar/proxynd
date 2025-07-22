#!/bin/bash

# Git 히스토리에서 tasks/ 디렉토리를 완전히 제거하는 스크립트

echo "⚠️  경고: 이 작업은 Git 히스토리를 다시 작성합니다!"
echo "백업을 먼저 만드시겠습니까? (y/n)"
read -r response

if [[ "$response" == "y" ]]; then
    echo "백업 생성 중..."
    git branch backup-before-tasks-removal
    echo "✅ 백업 브랜치 생성됨: backup-before-tasks-removal"
fi

echo ""
echo "다음 중 하나를 선택하세요:"
echo "1. git filter-branch 사용 (전통적 방법)"
echo "2. git-filter-repo 사용 (권장 - 더 빠르고 안전)"
echo "3. BFG Repo-Cleaner 사용 (가장 빠름)"
read -r method

case $method in
    1)
        echo "🔄 git filter-branch를 사용하여 tasks/ 디렉토리 제거 중..."

        # filter-branch를 사용하여 모든 커밋에서 tasks/ 디렉토리 제거
        git filter-branch --force --index-filter \
            'git rm -r --cached --ignore-unmatch tasks/' \
            --prune-empty --tag-name-filter cat -- --all

        echo "✅ filter-branch 완료"
        ;;

    2)
        echo "🔄 git-filter-repo를 사용하여 tasks/ 디렉토리 제거 중..."

        # git-filter-repo 설치 확인
        if ! command -v git-filter-repo &> /dev/null; then
            echo "git-filter-repo가 설치되어 있지 않습니다."
            echo "설치하시겠습니까? (y/n)"
            read -r install
            if [[ "$install" == "y" ]]; then
                # macOS
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    brew install git-filter-repo
                # Python pip
                else
                    pip install git-filter-repo
                fi
            else
                echo "설치를 취소합니다."
                exit 1
            fi
        fi

        # git-filter-repo 실행
        git filter-repo --path tasks/ --invert-paths --force

        echo "✅ git-filter-repo 완료"
        ;;

    3)
        echo "🔄 BFG Repo-Cleaner를 사용하여 tasks/ 디렉토리 제거 중..."

        # BFG 다운로드 확인
        if [[ ! -f "bfg.jar" ]]; then
            echo "BFG Repo-Cleaner 다운로드 중..."
            curl -L https://repo1.maven.org/maven2/com/madgag/bfg/1.14.0/bfg-1.14.0.jar -o bfg.jar
        fi

        # BFG 실행
        java -jar bfg.jar --delete-folders tasks --no-blob-protection .

        # 정리
        git reflog expire --expire=now --all && git gc --prune=now --aggressive

        echo "✅ BFG Repo-Cleaner 완료"
        ;;

    *)
        echo "잘못된 선택입니다."
        exit 1
        ;;
esac

# 공통 정리 작업
echo ""
echo "🧹 정리 작업 수행 중..."

# 오래된 참조 제거
rm -rf .git/refs/original/

# 가비지 컬렉션
git reflog expire --expire=now --all
git gc --prune=now --aggressive

# 결과 확인
echo ""
echo "📊 결과 확인:"
echo "남은 tasks/ 관련 커밋 수:"
git log --all --full-history -- tasks/ | grep -c "^commit" || echo "0"

echo ""
echo "✅ 작업 완료!"
echo ""
echo "⚠️  중요: 원격 저장소에 강제 푸시하려면:"
echo "git push origin --force --all"
echo "git push origin --force --tags"
echo ""
echo "⚠️  주의사항:"
echo "1. 다른 사람과 협업 중이라면 미리 알려야 합니다"
echo "2. 모든 로컬 복사본을 새로 클론해야 합니다"
echo "3. 이 작업은 되돌릴 수 없습니다 (백업 브랜치 제외)"
