package exiftool

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Exiftool is the exiftool utility wrapper
type Exiftool struct {
	mux sync.Mutex

	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdMergedOut  io.ReadCloser
	scanMergedOut *bufio.Scanner
}

// NewExiftool instanciates a new Exiftool with configuration functions. If anything went
// wrong, a non empty error will be returned.
func NewExiftool() (*Exiftool, error) {
	e := &Exiftool{}
	e.cmd = exec.Command(exiftoolBinary, "-stay_open", "True", "-@", "-")

	r, w := io.Pipe()
	e.stdMergedOut = r

	e.cmd.Stdout = w
	e.cmd.Stderr = w

	var err error
	if e.stdin, err = e.cmd.StdinPipe(); err != nil {
		return nil, err
	}

	e.scanMergedOut = bufio.NewScanner(e.stdMergedOut)
	e.scanMergedOut.Split(splitReadyToken)

	if err = e.cmd.Start(); err != nil {
		return nil, fmt.Errorf("error when executing command: %w", err)
	}

	return e, nil
}

func (e *Exiftool) Close() error {
	e.mux.Lock()
	defer e.mux.Unlock()

	for _, v := range []string{"-stay_open", "False"} {
		if _, err := fmt.Fprintln(e.stdin, v); err != nil {
			return err
		}
	}

	var err error
	if err = e.stdMergedOut.Close(); err != nil {
		return err
	}
	if err = e.stdin.Close(); err != nil {
		return err
	}

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		if e.cmd != nil {
			if err = e.cmd.Wait(); err != nil {
				return
			}
		}
	}()

	// Wait for wait to finish or timeout
	select {
	case <-ch:
	case <-time.After(time.Second):
		err = e.cmd.Process.Kill()
	}

	return err
}

func (e *Exiftool) Scan(path string) (string, error) {
	e.mux.Lock()
	defer e.mux.Unlock()

	for _, v := range []string{"-j", path, "-execute"} {
		if _, err := fmt.Fprintln(e.stdin, v); err != nil {
			return "", err
		}
	}

	ok := e.scanMergedOut.Scan()
	serr := e.scanMergedOut.Err()
	if serr != nil {
		return "", serr
	}
	if !ok {
		return "", errors.New("error while reading stdMergedOut: EOF")
	}
	res := strings.TrimSpace(e.scanMergedOut.Text())
	if strings.HasPrefix(res, "Error: ") {
		return "", errors.New(strings.ReplaceAll(res, "Error: ", ""))
	}
	return e.scanMergedOut.Text(), nil
}

var readyTokenLen = len(readyToken)

func splitReadyToken(data []byte, atEOF bool) (int, []byte, error) {
	idx := bytes.Index(data, readyToken)
	if idx == -1 {
		if atEOF && len(data) > 0 {
			return 0, data, fmt.Errorf("no final token found")
		}

		return 0, nil, nil
	}

	return idx + readyTokenLen, data[:idx], nil
}
