package api

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/glueops/apiary/internal/config"
)

//go:embed all:webdist
var webdist embed.FS

// apiFallthrough answers a request that reached the frontend fallback but was
// addressed to the API, and reports whether it did.
//
// The frontend is mounted on every unmatched path, so an API route that does not
// exist -- a typo, a client built against a newer server, a path retired between
// releases -- lands here rather than at the API. Neither fallback then does
// anything an API client can read. In embed mode a browser Accept header serves
// index.html with 200, so a missing endpoint reports success and returns HTML.
// In proxy mode it is reverse-proxied to Vite, whose own config proxies the API
// prefix back here, and the request rebounds between the two until something
// gives up: the observed symptom was a 502 for a route that simply does not
// exist.
//
// So the API prefix stops here, and it stops for every method and every Accept
// header, before either fallback is consulted.
func apiFallthrough(prefix string) func(http.ResponseWriter, *http.Request) bool {
	if prefix == "" {
		return func(http.ResponseWriter, *http.Request) bool { return false }
	}
	prefix = "/" + strings.Trim(prefix, "/")
	return func(w http.ResponseWriter, r *http.Request) bool {
		p := r.URL.Path
		if p != prefix && !strings.HasPrefix(p, prefix+"/") {
			return false
		}
		// Shaped like every other error this API returns, so a generated client
		// parses it instead of failing on the content type.
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"title":"Not Found","status":404,` +
			`"detail":"no API route matches this path"}` + "\n"))
		return true
	}
}

func WireFrontend(router interface {
	Handle(pattern string, h http.Handler)
	NotFound(h http.HandlerFunc)
}, mode string, env string, embeddedDist fs.FS, viteDevURL string, apiPrefix string) error {
	canonical, err := config.NormalizeFrontendMode(mode, env)
	if err != nil {
		return err
	}
	switch canonical {
	case config.FrontendModeProxy:
		return wireViteProxy(router, viteDevURL, apiPrefix)
	case config.FrontendModeEmbed:
		return wireEmbeddedSPA(router, embeddedDist, apiPrefix)
	case config.FrontendModeNone:
		return nil
	default:
		return fmt.Errorf("api: unhandled frontend mode %q", canonical)
	}
}

func wireViteProxy(router interface {
	NotFound(h http.HandlerFunc)
}, viteDevURL string, apiPrefix string) error {
	if viteDevURL == "" {
		viteDevURL = "http://localhost:5173"
	}

	target, err := url.Parse(viteDevURL)
	if err != nil {
		return err
	}
	if target.Scheme == "" || target.Host == "" {
		return errors.New("VITE_DEV_URL must include scheme and host, e.g. http://localhost:5173")
	}
	slog.Warn("vite dev proxy mounted on all unmatched paths", "target", target.String())

	proxy := &httputil.ReverseProxy{
		Rewrite: func(preq *httputil.ProxyRequest) {
			preq.SetURL(target)

			preq.Out.Header.Del("X-Forwarded-For")
			preq.Out.Header.Del("X-Forwarded-Host")
			preq.Out.Header.Del("X-Forwarded-Proto")
			preq.Out.Header.Del("Forwarded")

			preq.SetXForwarded()
			preq.Out.Host = target.Host
		},
	}

	notAnAPIRoute := apiFallthrough(apiPrefix)

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if notAnAPIRoute(w, r) {
			return
		}
		switch r.Method {
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
			return
		case http.MethodGet, http.MethodHead:

		default:
			http.NotFound(w, r)
			return
		}

		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "text/html") &&
			!strings.Contains(accept, "text/css") &&
			!strings.Contains(accept, "application/javascript") &&
			!strings.Contains(accept, "image/") &&
			accept != "*/*" {

			http.NotFound(w, r)
			return
		}
		proxy.ServeHTTP(w, r)
	})
	return nil
}

func wireEmbeddedSPA(router interface {
	NotFound(h http.HandlerFunc)
}, dist fs.FS, apiPrefix string) error {
	serveFSFile := func(w http.ResponseWriter, r *http.Request, p string) bool {
		name := strings.TrimPrefix(p, "/")

		f, err := dist.Open(name)
		if err != nil {
			return false
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return false
		}
		if info.IsDir() {
			return false
		}

		if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}

		if p == "/index.html" {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}

		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return true
		}

		w.Header().Set("Content-Length", "")
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, f)
		return true
	}

	notAnAPIRoute := apiFallthrough(apiPrefix)

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if notAnAPIRoute(w, r) {
			return
		}
		switch r.Method {
		case http.MethodOptions:
			w.WriteHeader(http.StatusNoContent)
			return
		case http.MethodGet, http.MethodHead:

		default:
			http.NotFound(w, r)
			return
		}

		p := path.Clean(r.URL.Path)
		if p == "/" {
			p = "/index.html"
		}

		if serveFSFile(w, r, p) {
			return
		}

		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/html") {

			if serveFSFile(w, r, "/index.html") {
				return
			}
		}

		http.NotFound(w, r)
	})
	return nil
}
