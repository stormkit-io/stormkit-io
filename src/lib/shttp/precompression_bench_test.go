package shttp_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/require"
)

// Compare the two modes side by side with:
//
//	go test ./src/lib/shttp -run '^$' -bench Precompression -benchmem -count 10 -p 1 > bench.txt
//	benchstat -col /mode bench.txt
var precompressionSizes = []struct {
	name string
	size int
}{
	{name: "30KB", size: 30 << 10},
	{name: "50KB", size: 50 << 10},
	{name: "300KB", size: 300 << 10},
}

type precompressionBench struct {
	b      *testing.B
	source []byte
}

type precompressionHandlerParams struct {
	body   []byte
	packed []byte
}

func newPrecompressionBench(b *testing.B) *precompressionBench {
	pb := &precompressionBench{b: b}
	pb.source = pb.loadSource()

	return pb
}

// loadSource concatenates the dashboard's TypeScript sources. Real code
// compresses very differently from generated filler, which would flatter
// both modes.
func (pb *precompressionBench) loadSource() []byte {
	var buf bytes.Buffer

	root := filepath.Join("..", "..", "ui", "src")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !(strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
			return nil
		}

		data, err := os.ReadFile(path)

		if err != nil {
			return err
		}

		buf.Write(data)

		return nil
	})

	require.NoError(pb.b, err)

	return buf.Bytes()
}

func (pb *precompressionBench) body(size int) []byte {
	if len(pb.source) < size {
		pb.b.Skipf("only %d bytes of source available, need %d", len(pb.source), size)
	}

	return pb.source[:size]
}

// pack compresses the way the file cache does when it stores a copy.
func (pb *precompressionBench) pack(body []byte) []byte {
	var buf bytes.Buffer

	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	require.NoError(pb.b, err)

	_, err = writer.Write(body)
	require.NoError(pb.b, err)
	require.NoError(pb.b, writer.Close())

	return buf.Bytes()
}

// handler mirrors the hosting router: the gzip middleware wraps both routes,
// one returning the plain body and one returning the stored copy.
func (pb *precompressionBench) handler(p precompressionHandlerParams) http.Handler {
	r := shttp.NewRouter()

	r.NewService().NewEndpoint("").
		Handler(http.MethodGet, "/per-request", func(*shttp.RequestContext) *shttp.Response {
			return &shttp.Response{
				Status:  http.StatusOK,
				Headers: http.Header{"Content-Type": []string{"application/javascript"}},
				Data:    p.body,
			}
		}).
		Handler(http.MethodGet, "/stored", func(*shttp.RequestContext) *shttp.Response {
			return &shttp.Response{
				Status: http.StatusOK,
				Headers: http.Header{
					"Content-Type":     []string{"application/javascript"},
					"Content-Encoding": []string{"gzip"},
					"Vary":             []string{"Accept-Encoding"},
				},
				Data: p.packed,
			}
		})

	return r.WithGzip().Handler()
}

func (pb *precompressionBench) request(path string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	return req
}

// verify fails the run if either mode is not doing what it is meant to, since
// a benchmark of a stored copy that gets compressed again would be meaningless.
func (pb *precompressionBench) verify(handler http.Handler, p precompressionHandlerParams) {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, pb.request("/per-request"))

	require.Equal(pb.b, "gzip", rec.Header().Get("Content-Encoding"))

	reader, err := gzip.NewReader(rec.Body)
	require.NoError(pb.b, err)

	decoded, err := io.ReadAll(reader)
	require.NoError(pb.b, err)
	require.Equal(pb.b, p.body, decoded)

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, pb.request("/stored"))

	require.Equal(pb.b, p.packed, rec.Body.Bytes(), "the stored copy must bypass the middleware")
}

// discardWriter counts what would go on the wire without buffering it, so a
// recorder's own copying does not blur the difference being measured.
type discardWriter struct {
	header  http.Header
	written int
}

func (w *discardWriter) Header() http.Header {
	return w.header
}

func (w *discardWriter) Write(p []byte) (int, error) {
	w.written += len(p)

	return len(p), nil
}

func (w *discardWriter) WriteHeader(int) {}

func (w *discardWriter) reset() {
	clear(w.header)
	w.written = 0
}

// BenchmarkPrecompression measures the cost of serving one response when the
// middleware compresses it versus when a stored copy is written through.
func BenchmarkPrecompression(b *testing.B) {
	pb := newPrecompressionBench(b)

	for _, sz := range precompressionSizes {
		body := pb.body(sz.size)
		params := precompressionHandlerParams{body: body, packed: pb.pack(body)}
		handler := pb.handler(params)

		pb.verify(handler, params)

		for _, mode := range []string{"per-request", "stored"} {
			b.Run("mode="+mode+"/size="+sz.name, func(b *testing.B) {
				req := pb.request("/" + mode)
				w := &discardWriter{header: http.Header{}}

				b.SetBytes(int64(len(body)))
				b.ReportAllocs()
				b.ResetTimer()

				for i := 0; i < b.N; i++ {
					w.reset()
					handler.ServeHTTP(w, req)
				}

				b.ReportMetric(float64(w.written), "wire-bytes")
			})
		}
	}
}

// BenchmarkPrecompressionParallel is the same comparison under concurrent
// load, where per-request compression competes with other requests for CPU.
func BenchmarkPrecompressionParallel(b *testing.B) {
	pb := newPrecompressionBench(b)

	for _, sz := range precompressionSizes {
		body := pb.body(sz.size)
		handler := pb.handler(precompressionHandlerParams{body: body, packed: pb.pack(body)})

		for _, mode := range []string{"per-request", "stored"} {
			b.Run("mode="+mode+"/size="+sz.name, func(b *testing.B) {
				b.SetBytes(int64(len(body)))
				b.ReportAllocs()
				b.ResetTimer()

				b.RunParallel(func(p *testing.PB) {
					req := pb.request("/" + mode)
					w := &discardWriter{header: http.Header{}}

					for p.Next() {
						w.reset()
						handler.ServeHTTP(w, req)
					}
				})
			})
		}
	}
}

// BenchmarkPrecompressionBuild is the one-off cost the stored mode pays on the
// first request for a file, at the highest compression level.
func BenchmarkPrecompressionBuild(b *testing.B) {
	pb := newPrecompressionBench(b)

	for _, sz := range precompressionSizes {
		body := pb.body(sz.size)

		b.Run("size="+sz.name, func(b *testing.B) {
			b.SetBytes(int64(len(body)))
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				pb.pack(body)
			}
		})
	}
}
