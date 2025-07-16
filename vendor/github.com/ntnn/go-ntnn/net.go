package ntnn

import (
	"io"
	"net"
	"net/http"
	"os"
)

func UnusedPort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	Panic(err)
	port := l.Addr().(*net.TCPAddr).Port
	Panic(l.Close())
	return port
}

func DumpToFile(addr, out string) {
	resp, err := http.Get(addr)
	Panic(err)
	defer Panic(resp.Body.Close())

	f, err := os.Create(out)
	Panic(err)
	defer Panic(f.Close())

	if _, err := io.Copy(f, resp.Body); err != nil {
		Errorf(err, "error writing request body to file")
	}
}
