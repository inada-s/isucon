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

	left := -1
	open := false
	for i := 0; i < len(body)-1; i++ {
		token := string(body[i]) + string(body[i+1])
		if token == leftDelim {
			left = i
			i++
			if open {
				w.Write([]byte("`)"))
				open = false
			}
		} else if token == rightDelim {
			i++
			w.Write([]byte("\n// "))
			w.Write(body[left : i+1])
			w.Write([]byte("\n"))
			left = -1
		} else if left == -1 {
			if token == "  " {
				continue
			}
			if body[i] == '\n' {
				continue
			}
			if !open {
				w.Write([]byte("iw.WriteString(`"))
				open = true
			}
			w.Write(body[i : i+1])
		}
	}
	if open {
		w.Write([]byte("`)\n"))
		open = false
	}
}

func main() {
	run(os.Stdin, os.Stdout)
}
