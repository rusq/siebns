package siebns

import (
	"bufio"
	"io"
	"strings"
)

type lineReader interface {
	readline() []byte
	readstring() string
	position() int64
	err() error
}

func newLineReader(r io.Reader) lineReader {
	return &lnReader{r: bufio.NewReader(r)}
}

type lnReader struct {
	r       *bufio.Reader
	pos     int64
	lastErr error
}

func (lr *lnReader) readline() (line []byte) {
	if lr.lastErr != nil {
		return
	}
	line, err := lr.r.ReadBytes('\n')
	if err != nil {
		lr.lastErr = err
		return
	}
	lr.pos += int64(len(line))

	return
}

func (lr *lnReader) readstring() (line string) {
	return strings.Trim(string(lr.readline()), string(crlf))
}

func (lr *lnReader) position() int64 {
	return lr.pos
}

func (lr *lnReader) err() error {
	return lr.lastErr
}
