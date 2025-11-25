package gin

import (
	"github.com/gin-gonic/gin"
	ginx "github.com/hopeio/gox/net/http/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Prom(r *gin.Engine) {
	// Register Metrics metrics handler.
	r.Any("/metrics", ginx.Wrap(promhttp.Handler()))
}
