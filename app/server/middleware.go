package server

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	log "github.com/go-pkgz/lgr"
)

// JSON is a map alias, just for convenience
type JSON map[string]interface{}

// LoggerFlag type
type LoggerFlag int

// logger flags enum
const (
	LogAll LoggerFlag = iota
	LogBody
)
const maxBody = 1024

var reMultWhtsp = regexp.MustCompile(`[\s\p{Zs}]{2,}`)

// Logger middleware prints http log. Customized by set of LoggerFlag
func Logger(l log.L, flags ...LoggerFlag) func(http.Handler) http.Handler {

	inFlags := func(f LoggerFlag) bool {
		for _, flg := range flags {
			if flg == LogAll || flg == f {
				return true
			}
		}
		return false
	}

	f := func(h http.Handler) http.Handler {

		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, 1)

			body := func() (result string) {
				if inFlags(LogBody) {
					if content, err := io.ReadAll(r.Body); err == nil {
						result = string(content)
						r.Body = io.NopCloser(bytes.NewReader(content))

						if len(result) > 0 {
							result = strings.Replace(result, "\n", " ", -1)
							result = reMultWhtsp.ReplaceAllString(result, " ")
						}

						if len(result) > maxBody {
							result = result[:maxBody] + "..."
						}
					}
				}
				return result
			}()

			t1 := time.Now()
			defer func() {
				t2 := time.Now()

				q := r.URL.String()
				if qun, err := url.QueryUnescape(q); err == nil {
					q = qun
				}
				l.Logf("[INFO] REST %s - %s - %s - %d (%d) - %v %s",
					r.Method, q, strings.Split(r.RemoteAddr, ":")[0],
					ww.Status(), ww.BytesWritten(), t2.Sub(t1), body)
			}()

			h.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}

	return f
}
