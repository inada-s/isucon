package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

func run(r io.Reader, w io.Writer) {
	const (
		leftDelim  = "{{"
		rightDelim = "}}"
	)
	body, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalln(err)
	}
	if len(body) == 0 {
		log.Println("no content")
	}

	var left int
	w.Write([]byte("iw.WriteString(`"))
	for i := 0; i < len(body)-1; i++ {
		token := string(body[i]) + string(body[i+1])
		if token == leftDelim {
			left = i
			i++
			w.Write([]byte("`)"))
		} else if token == rightDelim {
			i++
			w.Write([]byte("\n// "))
			w.Write(body[left : i+1])
			w.Write([]byte("\n"))
			left = 0
			w.Write([]byte("iw.WriteString(`"))
		} else if left == 0 {
			w.Write(body[i : i+1])
		}
	}
	w.Write([]byte("`)\n"))
}

func main() {
	run(os.Stdin, os.Stdout)
}
