package render

import (
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// PageDataContentType is the native client's versioned page-data contract.
// Ordinary web requests continue receiving the HTML shell at the same routes.
const PageDataContentType = "application/vnd.agrilinks.page.v1+json"

type pageDataWriter struct{ http.ResponseWriter }

func (*pageDataWriter) pageDataResponse()             {}
func (w *pageDataWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// PageDataRequests negotiates page snapshots without changing authentication,
// mutations, redirects, or existing JSON endpoints. RenderPage still projects
// only the fields explicitly allowed by pages.json.
func PageDataRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestsPageData(r) {
			next.ServeHTTP(&pageDataWriter{w}, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequestsPageData identifies the native page client for negotiated responses
// and its checkout return screen. It never grants authentication or permission.
func RequestsPageData(r *http.Request) bool {
	for _, value := range r.Header.Values("Accept") {
		for _, item := range strings.Split(value, ",") {
			mediaType, params, err := mime.ParseMediaType(strings.TrimSpace(item))
			if err != nil || mediaType != PageDataContentType {
				continue
			}
			if quality, exists := params["q"]; exists {
				q, err := strconv.ParseFloat(quality, 64)
				if err != nil || !(q > 0 && q <= 1) {
					continue
				}
			}
			return true
		}
	}
	return false
}

func wantsPageData(w http.ResponseWriter) bool {
	for {
		if _, ok := w.(interface{ pageDataResponse() }); ok {
			return true
		}
		wrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			return false
		}
		w = wrapper.Unwrap()
	}
}
