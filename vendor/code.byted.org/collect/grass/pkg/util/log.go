package util

import (
	"bytes"
	"log"

	"code.byted.org/gopkg/logfmt"
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func Log(keyvals ...interface{}) error {
	var buf bytes.Buffer
	e := logfmt.NewEncoder(&buf)
	e.EncodeKeyvals(keyvals...)
	e.EndRecord()
	// log.Printf("%s", buf.String())
	log.Output(2, buf.String())
	return nil
}
