// Package httpmw holds reusable net/http middlewares shared across apps.
package httpmw

import "net/http"

// SecurityHeaders adds baseline security response headers.
// CSP is intentionally not set here: front-end asset hosts vary per deployment
// and a wrong policy is costly, so leave it to the reverse proxy per site.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// HSTS only makes sense over HTTPS; when TLS terminates at the reverse
		// proxy r.TLS is nil here, so HSTS is the proxy's responsibility.
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
