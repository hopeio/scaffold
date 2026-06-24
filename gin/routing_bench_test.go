package gin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	gin "github.com/gin-gonic/gin"
)

const benchNumRoutes = 1000

// 多样化前缀模板，模拟真实 API 路由分布
var staticPatterns = []string{
	"/user/profile/%d",
	"/order/detail/%d",
	"/product/list/%d",
	"/admin/dashboard/%d",
	"/api/v1/auth/%d",
	"/api/v2/auth/%d",
	"/settings/account/%d",
	"/report/sales/%d",
	"/dashboard/widget/%d",
	"/billing/invoice/%d",
}

func genStaticPaths(n int) []string {
	paths := make([]string, n)
	patterns := staticPatterns
	for i := 0; i < n; i++ {
		paths[i] = fmt.Sprintf(patterns[i%len(patterns)], i)
	}
	return paths
}

func setupGinStaticRouter(paths []string) *gin.Engine {
	r := gin.New()
	for _, p := range paths {
		r.GET(p, func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
	}
	return r
}

func setupNetHTTPStaticRouter(paths []string) *http.ServeMux {
	mux := http.NewServeMux()
	for _, p := range paths {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
	return mux
}

// --- 参数路由 ---

// gin 参数路由模板
var ginParamRoutes = []string{
	"/user/:id",
	"/order/:orderId/items/:itemId",
	"/product/:category/:id",
	"/api/:version/users/:id",
	"/article/:slug/comments/:commentId",
	"/org/:orgId/team/:teamId/member/:memberId",
	"/store/:store/product/:productId",
	"/blog/:year/:month/:day",
}

// net/http (Go 1.22+) 参数路由模板
var nethttpParamRoutes = []string{
	"/user/{id}",
	"/order/{orderId}/items/{itemId}",
	"/product/{category}/{id}",
	"/api/{version}/users/{id}",
	"/article/{slug}/comments/{commentId}",
	"/org/{orgId}/team/{teamId}/member/{memberId}",
	"/store/{store}/product/{productId}",
	"/blog/{year}/{month}/{day}",
}

// 为每条参数路由生成一组测试路径（替换参数为具体值）
func genParamTestPaths(n int) []string {
	var paths []string
	for i := 0; i < n; i++ {
		v := fmt.Sprintf("%d", i)
		paths = append(paths,
			fmt.Sprintf("/user/%s", v),
			fmt.Sprintf("/order/%s/items/%s", v, v),
			fmt.Sprintf("/product/electronics/%s", v),
			fmt.Sprintf("/api/v1/users/%s", v),
			fmt.Sprintf("/article/hello-world/comments/%s", v),
			fmt.Sprintf("/org/acme/team/eng/member/%s", v),
			fmt.Sprintf("/store/shop1/product/%s", v),
			fmt.Sprintf("/blog/2025/06/%s", v),
		)
	}
	return paths
}

func setupGinParamRouter() *gin.Engine {
	r := gin.New()
	for _, route := range ginParamRoutes {
		r.GET(route, func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
	}
	return r
}

func setupNetHTTPParamRouter() *http.ServeMux {
	mux := http.NewServeMux()
	for _, route := range nethttpParamRoutes {
		mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
	return mux
}

// --- benchmark 公共逻辑 ---

func benchRouting(b *testing.B, handler http.Handler, paths []string) {
	n := len(paths)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, paths[i%n], nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("unexpected status %d for %s", w.Code, paths[i%n])
		}
	}
}

// --- 静态路由 benchmark（多样化前缀） ---

func BenchmarkGinStaticRouting(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	paths := genStaticPaths(benchNumRoutes)
	router := setupGinStaticRouter(paths)
	benchRouting(b, router, paths)
}

func BenchmarkNetHTTPStaticRouting(b *testing.B) {
	paths := genStaticPaths(benchNumRoutes)
	mux := setupNetHTTPStaticRouter(paths)
	benchRouting(b, mux, paths)
}

// --- 参数路由 benchmark ---

func BenchmarkGinParamRouting(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	router := setupGinParamRouter()
	paths := genParamTestPaths(50) // 8 条参数路由 × 50 组值 = 400 条测试路径
	benchRouting(b, router, paths)
}

func BenchmarkNetHTTPParamRouting(b *testing.B) {
	mux := setupNetHTTPParamRouter()
	paths := genParamTestPaths(50)
	benchRouting(b, mux, paths)
}

// --- 静态路由：首尾命中 ---

func BenchmarkGinStaticFirst(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	paths := genStaticPaths(benchNumRoutes)
	router := setupGinStaticRouter(paths)
	benchRouting(b, router, []string{paths[0]})
}

func BenchmarkNetHTTPStaticFirst(b *testing.B) {
	paths := genStaticPaths(benchNumRoutes)
	mux := setupNetHTTPStaticRouter(paths)
	benchRouting(b, mux, []string{paths[0]})
}

func BenchmarkGinStaticLast(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)
	paths := genStaticPaths(benchNumRoutes)
	router := setupGinStaticRouter(paths)
	benchRouting(b, router, []string{paths[len(paths)-1]})
}

func BenchmarkNetHTTPStaticLast(b *testing.B) {
	paths := genStaticPaths(benchNumRoutes)
	mux := setupNetHTTPStaticRouter(paths)
	benchRouting(b, mux, []string{paths[len(paths)-1]})
}
