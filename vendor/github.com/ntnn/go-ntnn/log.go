package ntnn

import (
	"fmt"
	"os"
	"runtime"
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

func printer(msg string) {
	if !EnableLogs {
		return
	}

	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}

	if LogToFile == "" {
		_, err := fmt.Print(Marker + " " + msg)
		Panic(err)
		return
	}

	fileLock.Lock()
	defer fileLock.Unlock()

	f, err := os.OpenFile(LogToFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	Panic(err)
	defer Panic(f.Close())

	_, err = f.WriteString(msg)
	Panic(err)
}

func Log(msg string) {
	printer(msg)
}

func Logf(format string, args ...any) {
	printer(fmt.Sprintf(format, args...))
}

func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		panic("error getting call stack")
	}
	return fmt.Sprintf("%s:%d", file, line)
}

var (
	lastLog     = map[string]string{}
	lastLogLock sync.RWMutex
)

func lastLogChanged(caller, msg string) bool {
	lastLogLock.RLock()
	defer lastLogLock.RUnlock()
	lastMsg, ok := lastLog[caller]
	return !ok || msg != lastMsg
}

func lastLogUpdate(caller, msg string) {
	lastLogLock.Lock()
	defer lastLogLock.Unlock()
	lastLog[caller] = msg
}

func logChanged(skip int, msg string) {
	c := caller(skip)
	if !lastLogChanged(c, msg) {
		return
	}
	printer(msg)
	lastLogUpdate(c, msg)
}

// LogChanged logs a message if the content has changed since the last
// invocation from the same source.
func LogChanged(msg string) {
	logChanged(2, msg)
}

// LogfChanged logs a formatted message if the content has changed since
// the last invocation from the same source.
func LogfChanged(format string, args ...any) {
	logChanged(2, fmt.Sprintf(format, args...))
}
