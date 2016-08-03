package main

import (
	"net/http"
	"time"

	"github.com/zenazn/goji/web"
	"github.com/zenazn/goji/web/middleware"
	"github.com/zenazn/goji/web/mutil"

	"github.com/reiki4040/rnlog"
)

const (
	LOG_REQUEST_ID     = "request_id"
	LOG_PROC_TIME_NANO = "proc_time_nano"
	LOG_PARAMS         = "reqest_params"
	LOG_HEADERS        = "reqest_headers"
)

func AccessLogger(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetReqID(*c)
		lw := mutil.WrapWriter(w)

		startTime := time.Now().UnixNano()
		h.ServeHTTP(lw, r)
		procTimeNano := time.Now().UnixNano() - startTime

		m := make(map[string]interface{}, 4)
		m[LOG_REQUEST_ID] = reqID
		m[LOG_PROC_TIME_NANO] = procTimeNano
		m[LOG_PARAMS] = r.Form
		m[LOG_HEADERS] = r.Header
		rnlog.Logging("ACCESS", "", m)
	}

	return http.HandlerFunc(fn)
}
