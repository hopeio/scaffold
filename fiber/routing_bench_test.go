package fiber

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	fiberx "github.com/gofiber/fiber/v3"
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

func setupFiberStaticRouter(paths []string) *fiberx.App {
	r := fiberx.New()
	for _, p := range paths {
		r.Get(p, func(c fiberx.Ctx) error {
			return c.SendStatus(http.StatusOK)
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

// fiber 参数路由模板
var fiberParamRoutes = []string{
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

func setupFiberParamRouter() *fiberx.App {
	r := fiberx.New()
	for _, route := range fiberParamRoutes {
		r.Get(route, func(c fiberx.Ctx) error {
			return c.SendStatus(http.StatusOK)
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

func benchFiberRouting(b *testing.B, app *fiberx.App, paths []string) {
	n := len(paths)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, paths[i%n], nil)
		resp, err := app.Test(req)
		if err != nil {
			b.Fatalf("fiber test error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			b.Fatalf("unexpected status %d for %s", resp.StatusCode, paths[i%n])
		}
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
}

func benchNetHTTPRouting(b *testing.B, handler http.Handler, paths []string) {
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

func BenchmarkFiberStaticRouting(b *testing.B) {
	paths := genStaticPaths(benchNumRoutes)
	router := setupFiberStaticRouter(paths)
	benchFiberRouting(b, router, paths)
}

func BenchmarkNetHTTPStaticRouting(b *testing.B) {
	paths := genStaticPaths(benchNumRoutes)
	mux := setupNetHTTPStaticRouter(paths)
	benchNetHTTPRouting(b, mux, paths)
}

// --- 参数路由 benchmark ---

func BenchmarkFiberParamRouting(b *testing.B) {
	router := setupFiberParamRouter()
	paths := genParamTestPaths(50)
	benchFiberRouting(b, router, paths)
}

func BenchmarkNetHTTPParamRouting(b *testing.B) {
	mux := setupNetHTTPParamRouter()
	paths := genParamTestPaths(50)
	benchNetHTTPRouting(b, mux, paths)
}

// --- 静态路由：首尾命中 ---

func BenchmarkFiberStaticFirst(b *testing.B) {
	paths := genStaticPaths(benchNumRoutes)
	router := setupFiberStaticRouter(paths)
	benchFiberRouting(b, router, []string{paths[0]})
}

func BenchmarkNetHTTPStaticFirst(b *testing.B) {
	paths := genStaticPaths(benchNumRoutes)
	mux := setupNetHTTPStaticRouter(paths)
	benchNetHTTPRouting(b, mux, []string{paths[0]})
}

func BenchmarkFiberStaticLast(b *testing.B) {
	paths := genStaticPaths(benchNumRoutes)
	router := setupFiberStaticRouter(paths)
	benchFiberRouting(b, router, []string{paths[len(paths)-1]})
}

func BenchmarkNetHTTPStaticLast(b *testing.B) {
	paths := genStaticPaths(benchNumRoutes)
	mux := setupNetHTTPStaticRouter(paths)
	benchNetHTTPRouting(b, mux, []string{paths[len(paths)-1]})
}
