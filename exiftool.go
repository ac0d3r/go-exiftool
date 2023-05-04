package exiftool

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"
)

var readyToken = []byte("{ready}")
var readyTokenLen = len(readyToken)

// Exiftool is the exiftool utility wrapper
type Exiftool struct {
	mux sync.Mutex

	cmd       *exec.Cmd
	stdin     io.WriteCloser
	mulStdout io.ReadCloser

	resReader bufio.Reader
	cache     *bytes.Buffer
}

// NewExiftool instanciates a new Exiftool with configuration functions. If anything went
// wrong, a non empty error will be returned.
func NewExiftool() (*Exiftool, error) {
	e := &Exiftool{}
	e.cmd = exec.Command(exiftoolBinary, "-stay_open", "True", "-@", "-")

	r, w := io.Pipe()
	e.mulStdout = r

	// merge stdout & stderr
	e.cmd.Stdout = w
	e.cmd.Stderr = w

	var err error
	if e.stdin, err = e.cmd.StdinPipe(); err != nil {
		return nil, err
	}

	e.resReader = *bufio.NewReader(e.mulStdout)
	e.cache = bytes.NewBuffer(make([]byte, 0, 1024))

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
	e.cache.Reset()
	if err = e.mulStdout.Close(); err != nil {
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

	for _, v := range []string{path, "-execute"} {
		if _, err := fmt.Fprintln(e.stdin, v); err != nil {
			return "", err
		}
	}

	e.cache.Reset()
	var (
		line []byte
		err  error
	)
	for {
		line, _, err = e.resReader.ReadLine()
		if err == io.EOF {
			break
		}
		if len(line) < readyTokenLen {
			goto Cache
		}
		if bytes.HasPrefix(line, readyToken) {
			break
		}
	Cache:
		e.cache.Write(append(line, '\n'))
	}
	if err != nil {
		return "", err
	}
	// 判断编码
	data := e.cache.Bytes()
	if IsGBK(data) {
		data, err = GbkToUtf8(data)
		if err != nil {
			return "", err
		}
	}

	res := strings.TrimSpace(string(data))
	if strings.HasPrefix(res, "Error: ") {
		return "", errors.New(strings.ReplaceAll(res, "Error: ", ""))
	}
	return parseOutput(res), nil
}

// parseOutput parses the output of exiftool and returns a json string
func parseOutput(res string) string {
	v := make(map[string]string)

	for _, line := range strings.Split(res, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		kv := strings.SplitN(line, ":", 2)
		if len(kv) != 2 {
			continue
		}

		k := strings.ReplaceAll(strings.TrimSpace(kv[0]), " ", "")
		if strings.Index(k, "/") > 0 {
			k = strings.ReplaceAll(k, "/Time", "")
		}
		v[k] = strings.TrimSpace(kv[1])
	}
	v["SourceFile"] = path.Join(v["Directory"], v["FileName"])
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}

	return string(data)
}
