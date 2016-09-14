package main

import (
	"fmt"
	"html"
	"io"
	"reflect"
	"unsafe"
)

type IsuWriter struct {
	io.Writer
}

// https://github.com/valyala/quicktemplate/blob/906c730a2e2f03b1f93eecb3f62f309f1bc29dc1/util_noappengine.go
func unsafeStrToBytes(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}

func (m IsuWriter) WriteString(s string) (n int, err error) {
	return m.Write(unsafeStrToBytes(s))
}

func (m IsuWriter) WriteEscString(s string) (n int, err error) {
	return m.WriteString(html.EscapeString(s))
}

func (m IsuWriter) Print(v interface{}) (n int, err error) {
	return m.WriteString(fmt.Sprint(v))
}
