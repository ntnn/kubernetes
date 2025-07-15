package ntnn

import (
	"bytes"
	"os"
	"runtime"
	"sync"
)

var dumpStackToFileLock sync.Mutex

func Stack() string {
	buf := make([]byte, 16*1024)
	runtime.Stack(buf, false)
	return string(bytes.Trim(buf, "\x00")) + "\n\n"
}

func DumpStackToFile(path, preStack, additionalInfo string) {
	curStack := Stack()

	dumpStackToFileLock.Lock()
	defer dumpStackToFileLock.Unlock()

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	Panic(err)
	defer f.Close()

	out := ""
	if preStack != "" {
		out += "preStack: " + preStack
	}

	if additionalInfo != "" {
		out += "additional: " + additionalInfo + "\n"
	}

	out += "curStack: " + curStack

	if _, err := f.WriteString(out); err != nil {
		Panic(err)
	}
}
