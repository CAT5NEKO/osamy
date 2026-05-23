package infrastructure

import (
	"net/http"
	"net/http/pprof"
	"time"
)

func NewPprofMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	return mux
}

func StartPprofServer(addr string) (*http.Server, error) {
	if addr == "" {
		return nil, nil
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           NewPprofMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		_ = server.ListenAndServe()
	}()

	return server, nil
}
