package ntnn

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

var (
	Marker     = "###>"
	EnableLogs = true

	fileLock  sync.Mutex
	LogToFile = ""
)

func init() {
	if LogToFile != "" {
		// Ensure the log file exists
		f, err := os.OpenFile(LogToFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		Panicf("error opening LogToFile", err)
		Panic(f.Close())
	}
}

func printer(format string, s ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}

	if LogToFile == "" {
		_, err := fmt.Printf(Marker+" "+format, s...)
		Panic(err)
		return
	}

	fileLock.Lock()
	defer fileLock.Unlock()

	f, err := os.OpenFile(LogToFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	Panic(err)
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf(format, s...))
	Panic(err)
}

func Log(s string) {
	if !EnableLogs {
		return
	}
	printer(s)
}

func Logf(format string, args ...any) {
	if !EnableLogs {
		return
	}
	printer(format, args...)
}
