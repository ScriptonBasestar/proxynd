package routers

import (
	"github.com/gin-gonic/gin"
	healthcheck "github.com/tavsec/gin-healthcheck"
	"github.com/tavsec/gin-healthcheck/checks"
	"github.com/tavsec/gin-healthcheck/config"
)

func HealthRouter(r *gin.Engine) {
	//confDirCheck := checks.NewEnvCheck("BASE_DIR")
	// You can also validate env format using regex
	//BaseDirCheck := checks.NewEnvCheck("BASE_DIR")
	//BaseDirCheck.SetRegexValidator("^(~|/).+")
	//healthcheck.New(r, config.DefaultConfig(), []checks.Check{confDirCheck})

	//ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	//defer stop()
	//signalsCheck := checks.NewContextCheck(ctx, "signals")
	//healthcheck.New(r, config.DefaultConfig(), []checks.Check{signalsCheck})
	healthcheck.New(r, config.DefaultConfig(), []checks.Check{})
}
