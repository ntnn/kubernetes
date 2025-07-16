package ntnn

import (
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"
)

func StartProfileServer(profile string) (string, func()) {
	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%d", UnusedPort()),
		Handler: pprof.Handler(profile),
	}
	go func() { Errorf(server.ListenAndServe(), "server exited") }()
	return server.Addr, func() { Errorf(server.Close(), "error closing http server") }
}

func StartProfileServerAndStall(profile string) {
	addr, stop := StartProfileServer(profile)
	defer stop()
	Logf("server available at: %q", addr)
	time.Sleep(99999 * time.Minute)
}

var DumpProfileTraceSeconds = "10"

func DumpProfile(profile, pathPrefix string) {
	Log("starting server to dump")
	addr, stop := StartProfileServer(profile)
	defer stop()

	pathPrefix += "-" + profile
	baseAddr := fmt.Sprintf("http://%s/debug/pprof/", addr)

	switch profile {
	case "goroutine":
		DumpToFile(baseAddr+profile+"?debug=0", pathPrefix+"-debug0.out")
		DumpToFile(baseAddr+profile+"?debug=2", pathPrefix+"-debug2.out")
	case "trace":
		DumpToFile(baseAddr+profile+"?seconds="+DumpProfileTraceSeconds, pathPrefix+".out")
	default:
		DumpToFile(baseAddr+profile, pathPrefix+".out")
	}
}
