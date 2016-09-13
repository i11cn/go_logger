package logger

import (
	"bytes"
)

type (
	Formatter interface {
		Format(Layout, [2]int, [5]string, ...interface{}) []byte
	}

	StringFormatter struct {
	}
)

func NewStringFormatter() *StringFormatter {
	return &StringFormatter{}
}

func (sf *StringFormatter) Format(layout Layout, int_args [2]int, str_args [5]string, args ...interface{}) []byte {
	var buf bytes.Buffer
	for _, l := range layout.Parts {
		buf.Write(l(int_args, str_args, args...))
	}
	return buf.Bytes()
}
