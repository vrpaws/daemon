//go:build !windows

package lib

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

func LogOutput(writer io.Writer) func() {
	logfile := fmt.Sprintf("log-%s.txt", time.Now().Format("2006-01-02_15-04-05"))
	f, _ := os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)

	mw := io.MultiWriter(writer, f)
	r, w := io.Pipe()

	log.SetOutput(mw)

	exit := make(chan struct{})
	go func() {
		_, _ = io.Copy(mw, r)
		close(exit)
	}()

	return func() {
		_ = w.Close()
		<-exit
		_ = f.Close()
	}
}

func DisableQuickEdit() {
	log.Println("Quick edit is not supported on Linux")
}
