package generator

import (
	"bytes"
	"fmt"
	"strings"
)

type indentWriter struct {
	b     bytes.Buffer
	level int
}

func (w *indentWriter) writef(tpl string, args ...interface{}) {
	indent := strings.Repeat("\t", w.level)
	for _, line := range strings.Split(fmt.Sprintf(tpl, args...), "\n") {
		w.b.WriteString(indent)
		w.b.WriteString(line)
	}
}

func (w *indentWriter) writefln(tpl string, args ...interface{}) {
	indent := strings.Repeat("\t", w.level)
	for _, line := range strings.Split(fmt.Sprintf(tpl, args...), "\n") {
		w.b.WriteString(indent)
		w.b.WriteString(line)
	}
	w.b.WriteRune('\n')
}

func (w *indentWriter) writeln(tpl string) {
	indent := strings.Repeat("\t", w.level)
	for _, line := range strings.Split(tpl, "\n") {
		w.b.WriteString(indent)
		w.b.WriteString(line)
	}
	w.b.WriteRune('\n')
}

func (w *indentWriter) write(tpl string) {
	indent := strings.Repeat("\t", w.level)
	for _, line := range strings.Split(tpl, "\n") {
		w.b.WriteString(indent)
		w.b.WriteString(line)
	}
}

func (w *indentWriter) in() {
	w.level++
}

func (w *indentWriter) out() {
	w.level--
}

func (w *indentWriter) String() string {
	return w.b.String()
}
