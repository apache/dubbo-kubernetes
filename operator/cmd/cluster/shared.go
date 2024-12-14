package cluster

import (
	"fmt"
	"io"
	"strings"
)

type writerPrinter struct {
	writer io.Writer
}

type Printer interface {
	Printf(string, ...any)
	Println(string)
}

func NewPtrForWtr(w io.Writer) Printer {
	return &writerPrinter{writer: w}
}

func (w writerPrinter) Printf(format string, a ...any) {
	_, _ = fmt.Fprintf(w.writer, format, a...)
}

func (w writerPrinter) Println(s string) {
	_, _ = fmt.Fprintln(w.writer, s)
}

func OptionDeterminator(msg string, writer io.Writer) bool {
	for {
		_, _ = fmt.Fprintf(writer, "%s ", msg)
		var resp string
		_, err := fmt.Scanln(&resp)
		if err != nil {
			return false
		}
		switch strings.ToUpper(resp) {
		case "Y", "YES", "y", "yes":
			return true
		case "N", "NO", "n", "no":
			return false
		}
	}
}

func NewPrinterForWriter(w io.Writer) Printer {
	return &writerPrinter{writer: w}
}
