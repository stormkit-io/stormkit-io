package rediscache_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
)

// Wordings for the same condition. go-redis classifies a read-only reply by
// prefix (see isReadOnlyError / isBadConn in the driver), so only the native
// one is recognised: it closes the connection and retries. A proxy that
// prefixes the reply with "ERR " defeats both checks.
const (
	nativeReadOnly = "-READONLY You can't write against a read only replica.\r\n"
	proxyReadOnly  = "-ERR READONLY You can't write against a read only instance.\r\n"
)

// stubRedis is a minimal RESP server that models a managed primary-standby
// switchover.
//
// Each connection records the generation it was accepted in. Failover bumps
// the generation, after which every connection opened before it answers writes
// as read-only forever, while connections opened after it are writable. That
// is the shape of the real event: the endpoint keeps resolving, new
// connections reach the promoted primary, and connections still pinned to the
// demoted node never recover on their own.
type stubRedis struct {
	ln         net.Listener
	wording    atomic.Value // string
	generation atomic.Int64
	conns      atomic.Int64
	holding    atomic.Bool
}

func newStubRedis(wording string) (*stubRedis, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")

	if err != nil {
		return nil, err
	}

	s := &stubRedis{ln: ln}
	s.wording.Store(wording)

	go s.accept()

	return s, nil
}

func (s *stubRedis) Addr() string {
	return s.ln.Addr().String()
}

// Failover promotes a new primary. Connections opened before this point are
// now talking to a demoted node and will only ever refuse writes.
func (s *stubRedis) Failover() {
	s.generation.Add(1)
}

// HoldReadOnly makes every connection refuse writes, new ones included, which
// is what a switchover looks like while it is still in progress rather than
// at the instant it completes.
func (s *stubRedis) HoldReadOnly() {
	s.holding.Store(true)
}

// Conns is the number of connections accepted since start. A pool that
// discards pinned connections and redials shows up here as an increase.
func (s *stubRedis) Conns() int64 {
	return s.conns.Load()
}

func (s *stubRedis) Close() {
	s.ln.Close()
}

func (s *stubRedis) accept() {
	for {
		conn, err := s.ln.Accept()

		if err != nil {
			return
		}

		s.conns.Add(1)

		go s.handle(conn, s.generation.Load())
	}
}

var writeCommands = map[string]bool{
	"SET": true, "SETEX": true, "DEL": true, "INCR": true,
	"LPUSH": true, "RPUSH": true, "LPOP": true, "GETDEL": true,
	"EXPIRE": true, "PEXPIRE": true,
}

func (s *stubRedis) handle(conn net.Conn, acceptedAt int64) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		args, err := readCommand(reader)

		if err != nil {
			return
		}

		if len(args) == 0 {
			continue
		}

		cmd := strings.ToUpper(args[0])
		demoted := s.holding.Load() || acceptedAt < s.generation.Load()

		var reply string

		switch {
		// Refuse the RESP3 handshake so the client falls back to RESP2,
		// which keeps this stub small.
		case cmd == "HELLO":
			reply = "-ERR unknown command 'HELLO'\r\n"
		case cmd == "PING":
			reply = "+PONG\r\n"
		case cmd == "GET":
			reply = "$-1\r\n"
		case writeCommands[cmd] && demoted:
			reply = s.wording.Load().(string)
		default:
			reply = "+OK\r\n"
		}

		if _, err := io.WriteString(conn, reply); err != nil {
			return
		}
	}
}

// readCommand parses one inline or array-form RESP command.
func readCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')

	if err != nil {
		return nil, err
	}

	line = strings.TrimRight(line, "\r\n")

	if line == "" {
		return nil, nil
	}

	if line[0] != '*' {
		return strings.Fields(line), nil
	}

	count, err := strconv.Atoi(line[1:])

	if err != nil {
		return nil, fmt.Errorf("bad multibulk header %q", line)
	}

	args := make([]string, 0, count)

	for range count {
		header, err := r.ReadString('\n')

		if err != nil {
			return nil, err
		}

		header = strings.TrimRight(header, "\r\n")

		if len(header) == 0 || header[0] != '$' {
			return nil, fmt.Errorf("bad bulk header %q", header)
		}

		size, err := strconv.Atoi(header[1:])

		if err != nil {
			return nil, err
		}

		buf := make([]byte, size+2)

		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}

		args = append(args, string(buf[:size]))
	}

	return args, nil
}
