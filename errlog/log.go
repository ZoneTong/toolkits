package errlog

import (
	"fmt"
	"io"
	"os"
)

type errlog struct {
	io.Writer
}

func (l *errlog) Log(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.Writer.Write([]byte(s + "\n"))
}

func (l *errlog) Logf(f string, v ...interface{}) {
	s := fmt.Sprintf(f, v...)
	l.Writer.Write([]byte(s + "\n"))
}

func New(w io.Writer) *errlog {
	return &errlog{w}
	// writer:= bufio.NewWriter(w)

}

var (
	DefaultLogger *errlog
)

func init() {
	DefaultLogger = New(os.Stderr)
}

func Log(v ...interface{}) {
	DefaultLogger.Log(v...)
}

func Logf(f string, v ...interface{}) {
	DefaultLogger.Logf(f, v...)
}
