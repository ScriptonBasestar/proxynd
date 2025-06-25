package middlewares

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"log"
	"os"
	"path/filepath"
	"time"
)

// AccessLogMiddleware 액세스 로그 미들웨어
// 모든 프록시 요청을 access.log 파일에 기록
func AccessLogMiddleware() fiber.Handler {
	// 로그 디렉토리 생성
	logDir := "./logs"
	os.MkdirAll(logDir, os.ModePerm)
	
	// 로그 파일 열기
	logFile, err := os.OpenFile(filepath.Join(logDir, "access.log"), 
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open access log file: %v", err)
	}
	
	return func(c *fiber.Ctx) error {
		// 요청 시작 시간
		start := time.Now()
		
		// 다음 핸들러 실행
		err := c.Next()
		
		// 요청 처리 시간 계산
		latency := time.Since(start)
		
		// 캐시 정보 가져오기
		cacheHit, _ := c.Locals("cache_hit").(bool)
		cacheStatus := "MISS"
		if cacheHit {
			cacheStatus = "HIT"
		}
		
		// 로그 포맷: [시간] IP 메소드 경로 상태코드 처리시간 캐시상태
		logEntry := fmt.Sprintf("[%s] %s %s %s %d %v %s\n",
			start.Format("2006-01-02 15:04:05"),
			c.IP(),
			c.Method(),
			c.Path(),
			c.Response().StatusCode(),
			latency,
			cacheStatus,
		)
		
		// 콘솔에 출력
		log.Print(logEntry)
		
		// 파일에 기록
		if logFile != nil {
			logFile.WriteString(logEntry)
		}
		
		return err
	}
}