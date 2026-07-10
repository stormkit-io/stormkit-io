package runner

import (
	"bytes"
	"regexp"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

type CustomBuffer struct {
	output          []byte
	ansiRegexp      *regexp.Regexp
	mu              sync.Mutex
	isStormkitCloud bool
}

func NewCustomBuffer() *CustomBuffer {
	return &CustomBuffer{
		ansiRegexp:      regexp.MustCompile(ansi),
		isStormkitCloud: config.IsStormkitCloud(),
	}
}

func (cb *CustomBuffer) Write(p []byte) (n int, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	input := cb.ansiRegexp.ReplaceAll(p, []byte(""))

	if cb.isStormkitCloud {
		input = bytes.Replace(input, []byte("/home/runner/work/deployer-service/deployer-service"), []byte("/home/app"), -1)
	}

	lines := bytes.Split(input, []byte("\n"))

	// Clean carriage returns
	for _, line := range lines {
		if crIndex := bytes.LastIndex(line, []byte("\r")); crIndex > -1 {
			line = line[crIndex+1:]
		}

		if len(line) == 0 || (len(line) <= 2 && line[0] == '/') {
			continue
		}

		line = append(line, []byte("\n")...)
		line = bytes.ReplaceAll(line, []byte("\x00"), []byte("")) // Remove null bytes
		cb.output = append(cb.output, line...)
	}

	// We always need to return the length of `p`:
	// See https://www.reddit.com/r/golang/comments/2xufjb/cmdrun_returning_short_write_as_error_why_does
	return len(p), nil
}

func (cb *CustomBuffer) String() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	return string(cb.output)
}
