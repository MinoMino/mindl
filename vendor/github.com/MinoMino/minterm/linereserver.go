package minterm

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Should be way easier to implement had it not been by the fact that
// os.Stdout/err aren't interfaces but *os.File. Hopefully Go 2 fixes it:
// https://github.com/golang/go/issues/13473

// Reserves a line at the bottom of the terminal, while normal stdout and
// stderr goes above it. It does not play well any other prints using
// carriage returns.
//
// A side-effect of the implementation is that anything printed that doesn't
// have a newline in it gets buffered instead of printed immediately.
type LineReserver struct {
	line            string
	out, err        *os.File
	r, w            *os.File
	flushChan       chan struct{}
	wait, flushWait sync.WaitGroup
	m               sync.Mutex
}

// Takes control of stdout and stderr in order to reserve the last line of the terminal,
// which can be set with Set().
func NewLineReserver() (*LineReserver, error) {
	// Make sure ahead of time nothing weird happens when we get terminal size.
	if _, _, err := TerminalSize(); err != nil {
		return nil, err
	}

	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	lr := &LineReserver{
		r:         r,
		w:         w,
		out:       os.Stdout,
		err:       os.Stderr,
		flushChan: make(chan struct{}),
	}
	lr.wait.Add(1)
	go lr.monitor()
	os.Stdout = w
	os.Stderr = w

	return lr, nil
}

// Clears the reserved line and restores control to stdout and stderr.
func (lr *LineReserver) Release() {
	lr.w.Close()
	lr.wait.Wait()
	os.Stdout = lr.out
	os.Stderr = lr.err
	lr.w = nil
}

// Sets the reserved line to the desired string.
func (lr *LineReserver) Set(line string) {
	lr.m.Lock()
	lr.line = line
	lr.m.Unlock()
}

// Prints the reserved line again, updating the line if it was changed
// since last time. Note that if something was buffered (i.e. something
// printed without a newline in it), a newline will be appended to avoid
// erasing what was in the buffer.
func (lr *LineReserver) Refresh() {
	if lr.w == nil {
		return
	}
	lr.flushWait.Add(1)
	lr.flushChan <- struct{}{}
	lr.flushWait.Wait()
}

func (lr *LineReserver) monitor() {
	defer lr.wait.Done()
	c := make(chan []byte)
	done := make(chan struct{})
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := lr.r.Read(buf)
			if err == io.EOF {
				done <- struct{}{}
				lr.r.Close()
				return
			}
			outbuf := make([]byte, n)
			copy(outbuf, buf[:n])
			c <- outbuf
		}
	}()

	var buf bytes.Buffer
	for {
		select {
		case b := <-c:
			buf.Write(b)
			// Only flush if we got a newline.
			if i := bytes.IndexByte(b, '\n'); i != -1 {
				lr.printLine(&buf)
			}
		case <-lr.flushChan:
			// We were told to flush.
			lr.printLine(&buf)
			lr.flushWait.Done()
		case <-done:
			lr.clearLine()
			buf.WriteTo(lr.out)
			return
		}
	}
}

func (lr *LineReserver) printLine(b *bytes.Buffer) {
	// We checked ahead that we can get the size, so we discard the error.
	// Panic not really an option here, but perhaps some other way of handling
	// potential errors should be done here. TODO.
	cols, _, _ := TerminalSize()
	// Check if the buffer has anything.
	var bs string
	if b.Len() != 0 {
		// We'd end up erasing stuff on the terminal if it doesn't end
		// on a newline, so we make sure to add one if there isn't.
		bs = ensureSuffix(b.String(), "\n")
	}
	lr.m.Lock()
	out := []byte(fmt.Sprintf("\r%s\r%s%s\r",
		strings.Repeat(" ", cols-1), bs, lr.line))
	lr.m.Unlock()
	lr.out.Write(out)
	b.Reset()
}

func (lr *LineReserver) clearLine() {
	cols, _, _ := TerminalSize()
	lr.out.Write([]byte(fmt.Sprintf("\r%s\r", strings.Repeat(" ", cols-1))))
}

func ensureSuffix(s, suffix string) string {
	if !strings.HasSuffix(s, suffix) {
		return s + suffix
	}

	return s
}
