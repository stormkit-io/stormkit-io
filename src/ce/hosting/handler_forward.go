package hosting

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/ce/api/accesslog"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v3"
)

const MAX_IMAGE_VARIANTS = 5
const SESSION_COOKIE_NAME = "stormkit_session"

var stormkitServerHeaderOff = os.Getenv("STORMKIT_SERVER_HEADER") == "off"

// HandlerForward forwards all requests. The return value is named so that the
// deferred finalize() can post-process every exit path (middleware
// short-circuit, NotFound, error, or the main Handle() flow).
func HandlerForward(req *RequestContext) (res *shttp.Response) {
	rs := NewRequestServer(req)

	defer func() {
		// Finalize the response (custom headers, snippets, analytics) so the
		// same transforms apply uniformly regardless of which path produced it.
		res = rs.finalize(res)

		// Send artifacts to redis queue with the finalized response visible. The
		// wait group lets tests (and graceful shutdown) drain in-flight pushes
		// instead of racing the global Batcher.
		artifactsWG.Add(1)

		go func() {
			defer artifactsWG.Done()
			rs.artifacts(res)
		}()
	}()

	slog.Debug(slog.LogOpts{
		Msg:     "handler forward received message",
		Level:   slog.DL4,
		Payload: rs.req.Fields,
	})

	if rs.req.Host == nil || rs.req.Host.Config == nil {
		return rs.NotFound()
	}

	middlewares := []func(req *RequestContext) (*shttp.Response, error){
		WithAnalyticsScript,
		WithCollect,
		WithSKAuth,
		WithAuthWall,
		WithRedirect,
	}

	for _, md := range middlewares {
		mres, err := md(req)

		if err != nil {
			return rs.Error(err)
		}

		if mres != nil {
			// We still want to be able to serve custom 404 pages
			// when the proxy request returns 404.
			if mres.Status == http.StatusNotFound {
				return rs.NotFound()
			}

			return mres
		}
	}

	slog.Debug(slog.LogOpts{
		Msg:     "middlewares complete",
		Level:   slog.DL4,
		Payload: rs.req.Fields,
	})

	return rs.Handle()
}

// finalize applies post-handler transforms — custom headers, snippet
// injection, analytics — uniformly to every response path.
func (r *RequestServer) finalize(res *shttp.Response) *shttp.Response {
	if res == nil || r.req.Host == nil || r.req.Host.Config == nil {
		return res
	}

	res = injectHeaders(r.req, res)
	res = injectSnippets(r.req, res)

	if r.req.Host.Config.IsEnterprise {
		contentType := strings.ToLower(res.Headers.Get("Content-Type"))

		if strings.HasPrefix(contentType, "text/html") {
			r.record = analyticsRecord(r.req, res)
		}
	}

	return res
}

type FileMeta struct {
	Name    string
	Headers map[string]string
}

type RequestServer struct {
	req       *RequestContext
	res       *shttp.Response
	client    integrations.ClientInterface
	cache     *rediscache.RedisCache
	fileMeta  *FileMeta
	imgName   string
	logs      []integrations.Log
	record    *analytics.Record
	fnInvoked bool
}

func NewRequestServer(req *RequestContext) *RequestServer {
	r := &RequestServer{
		req:    req,
		cache:  rediscache.Client(),
		client: integrations.Client(),
	}

	return r
}

func (r *RequestServer) artifacts(res *shttp.Response) {
	var data []byte

	if res == nil || r.req == nil || r.req.Host == nil || r.req.Host.Config == nil {
		return
	}

	if res.Data != nil {
		data, _ = res.Data.([]byte)
	}

	bandwidth := int64(len(data)) + headersSize(res.Headers)

	Queue(&jobs.HostingRecord{
		AppID:           r.req.Host.Config.AppID,
		EnvID:           r.req.Host.Config.EnvID,
		DeploymentID:    r.req.Host.Config.DeploymentID,
		HostName:        r.req.Host.Name,
		BillingUserID:   r.req.Host.Config.BillingUserID,
		FunctionInvoked: r.fnInvoked,
		Logs:            r.logs,
		Analytics:       r.record,
		AccessLog:       accessLogRecord(r.req, res, bandwidth),
		TotalBandwidth:  bandwidth,
	})
}

// accessLogRecord builds a raw access log for every request — including bots,
// XHR, static assets and non-HTML responses. Unlike analyticsRecord it applies
// no filtering and keeps the unmasked client IP and full user-agent; IsBot is
// recorded as a flag so bot traffic can be included or excluded at query time.
func accessLogRecord(req *RequestContext, res *shttp.Response, bytesSent int64) *accesslog.AccessLog {
	if req == nil || req.Host == nil || req.Host.Config == nil {
		return nil
	}

	userAgent := req.UserAgent()

	return &accesslog.AccessLog{
		AppID:        req.Host.Config.AppID,
		EnvID:        req.Host.Config.EnvID,
		DeploymentID: req.Host.Config.DeploymentID,
		DomainID:     req.Host.Config.DomainID,
		HostName:     req.Host.Name,
		RequestTS:    utils.NewUnix(),
		Method:       req.Method,
		RequestPath:  req.OriginalPath,
		StatusCode:   res.Status,
		ClientIP:     req.RemoteIP(),
		UserAgent:    userAgent,
		Referrer:     req.Referer(),
		IsBot:        analytics.IsBot(userAgent),
		BytesSent:    bytesSent,
		RequestID:    null.NewString(req.RequestID, req.RequestID != ""),
	}
}

func (r *RequestServer) FileMeta() *FileMeta {
	if len(r.req.Host.Config.StaticFiles) == 0 {
		return nil
	}

	requestPath := strings.ToLower(r.req.URL().Path)

	lookup := []string{
		requestPath,
		fmt.Sprintf("%s.html", requestPath),
		path.Join(requestPath, "index.html"),
	}

	for _, fileName := range lookup {
		if meta := r.req.Host.Config.StaticFiles[fileName]; meta != nil {
			return &FileMeta{
				Name:    meta.FileName,
				Headers: meta.Headers,
			}
		}
	}

	return nil
}

func (r *RequestServer) Handle() *shttp.Response {
	if r.fileMeta = r.FileMeta(); r.fileMeta != nil {
		return r.Static()
	}

	return r.Dynamic()
}

func (r *RequestServer) Static() *shttp.Response {
	notModified := false
	headers := shttp.HeadersFromMap(r.fileMeta.Headers)
	modifiedSinceHeader := r.req.Header.Get("If-Modified-Since")

	slog.Debug(slog.LogOpts{
		Msg:     "static handler initiated",
		Level:   slog.DL4,
		Payload: r.req.Fields,
	})

	// Check If-Modified-Since header -- give this priority
	if modifiedSinceHeader != "" && r.req.Host.Config.UpdatedAt.Valid {
		modifiedSinceTime, err := time.Parse(http.TimeFormat, modifiedSinceHeader)

		if err == nil {
			lastModifiedTime := time.Date(
				r.req.Host.Config.UpdatedAt.Time.Year(),
				r.req.Host.Config.UpdatedAt.Time.Month(),
				r.req.Host.Config.UpdatedAt.Time.Day(),
				r.req.Host.Config.UpdatedAt.Time.Hour(),
				r.req.Host.Config.UpdatedAt.Time.Minute(),
				r.req.Host.Config.UpdatedAt.Time.Second(),
				0,
				time.UTC,
			)

			notModified = !lastModifiedTime.After(modifiedSinceTime) || lastModifiedTime.Equal(modifiedSinceTime)
		}
	}

	// If-None-Match headers is checked only if the check above is not performed
	if noneMatchHeader := r.req.Header.Get("If-None-Match"); noneMatchHeader != "" && modifiedSinceHeader == "" {
		notModified = headers.Get("ETag") == noneMatchHeader
	}

	if headers.Get("Cache-Control") == "" {
		if strings.HasPrefix(headers.Get("Content-Type"), "text/html") {
			headers.Add("Cache-Control", "no-cache, must-revalidate")
		} else {
			headers.Add("Cache-Control", "public, max-age=86400")
		}
	}

	if headers.Get("Last-Modified") == "" && r.req.Host.Config.UpdatedAt.Valid {
		headers.Add("Last-Modified", r.req.Host.Config.UpdatedAt.Time.UTC().Format(http.TimeFormat))
	}

	if notModified {
		r.res = &shttp.Response{
			Status:  http.StatusNotModified,
			Headers: headers,
		}

		return r.res
	}

	var content []byte
	var err error

	content, err = r.fileContent(headers)

	if err != nil {
		return r.Error(err)
	}

	r.res = &shttp.Response{
		Status:  http.StatusOK,
		Data:    content,
		Headers: headers,
	}

	return r.res
}

func (r *RequestServer) Dynamic() *shttp.Response {
	cnf := r.req.Host.Config
	url := r.req.URL()
	arn := utils.GetString(cnf.FunctionLocation, cnf.APILocation)

	if arn == "" {
		return r.NotFound()
	}

	// If the path prefix is set and the request URL matches the prefix,
	// we need to route the request to the API location instead.
	if cnf.APILocation != "" && cnf.APIPathPrefix != "" && strings.HasPrefix(url.Path, cnf.APIPathPrefix) {
		arn = cnf.APILocation
	}

	// When snippets are configured we want the origin to hand back uncompressed
	// HTML so it can be injected into. Ask the upstream not to compress: if it
	// honours the header (most servers do) injectSnippets works on the plain
	// body directly; if it ignores it, injectSnippets still decompresses as a
	// fallback. Either way the edge gzip middleware re-compresses the final
	// response for the client.
	headers := r.req.Headers()

	if cnf.Snippets != nil {
		headers = headers.Clone()
		headers.Set("Accept-Encoding", "identity")
	}

	result, err := integrations.Client().Invoke(integrations.InvokeArgs{
		URL:           url,
		ARN:           arn,
		Body:          r.req.Body,
		ContentLength: r.req.ContentLength,
		Method:        r.req.Method,
		Headers:       headers,
		HostName:      r.req.Host.Name,
		AppID:         cnf.AppID,
		EnvID:         cnf.EnvID,
		DeploymentID:  cnf.DeploymentID,
		Command:       cnf.ServerCmd,
		EnvVariables:  cnf.EnvVariables,
		IsPublished:   cnf.Percentage > 0,
		CaptureLogs:   true,
		RemoteAddress: r.req.RemoteIP(),
		RemotePort:    r.req.RemotePort(),
		QueueLog: func(log *integrations.Log) {
			Queue(&jobs.HostingRecord{
				AppID:         r.req.Host.Config.AppID,
				EnvID:         r.req.Host.Config.EnvID,
				DeploymentID:  r.req.Host.Config.DeploymentID,
				HostName:      r.req.Host.Name,
				BillingUserID: r.req.Host.Config.BillingUserID,
				Logs:          []integrations.Log{*log},
			})
		},
		Context: map[string]any{
			"apiPrefix": cnf.APIPathPrefix,
		},
	})

	r.fnInvoked = true

	if result != nil && len(result.Logs) > 0 {
		r.logs = result.Logs
	}

	if err != nil {
		return r.Error(err)
	}

	if result == nil {
		return shttp.NoContent()
	}

	if result.ErrorMessage != "" && result.StatusCode == 0 {
		result.StatusCode = http.StatusInternalServerError
		result.Body = []byte(result.ErrorMessage)
	}

	r.res = &shttp.Response{
		Data:    result.Body,
		Status:  result.StatusCode,
		Headers: result.Headers,
	}

	return r.res
}

func (r *RequestServer) Error(requestErr error) *shttp.Response {
	fields := r.req.Fields
	fields = append(fields, zap.String("error", requestErr.Error()))

	slog.Debug(slog.LogOpts{
		Msg:     "request error",
		Level:   slog.DL4,
		Payload: fields,
	})

	cnf := r.req.Host.Config
	r.res = &shttp.Response{
		Status: http.StatusInternalServerError,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type": "text/html",
		}),
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Error",
			PageContent: html.Templates["error"],
			ContentData: map[string]any{
				"error_msg":        requestErr.Error(),
				"runtime_logs_url": admin.MustConfig().RuntimeLogsURL(cnf.AppID, cnf.EnvID, cnf.DeploymentID),
			},
		}),
	}

	customErrorFile := ErrorFile(cnf)

	if customErrorFile == nil {
		return r.res
	}

	file, err := r.client.GetFile(integrations.GetFileArgs{
		Location:     cnf.StorageLocation,
		FileName:     customErrorFile.FileName,
		DeploymentID: cnf.DeploymentID,
	})

	if err != nil || file == nil {
		return r.res
	}

	r.res.Data = file.Content
	r.res.Headers = shttp.HeadersFromMap(customErrorFile.Headers)
	r.res.Headers.Set("Content-Type", file.ContentType)

	return r.res
}

// NotFoundBuiltIn returns a built-in 404 page response.
// This is used when no custom 404 page is configured.
// It renders a simple HTML page with a 404 message and a link to the apps list.
func (r *RequestServer) NotFoundBuiltIn() *shttp.Response {
	r.res = &shttp.Response{
		Status: http.StatusNotFound,
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Page Not Found",
			PageContent: html.Templates["404"],
			ContentData: map[string]any{
				"app_url": admin.MustConfig().AppURL("/"),
			},
		}),
		Headers: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
		},
	}

	return r.res
}

func (r *RequestServer) NotFound() *shttp.Response {
	if r.req.Host == nil || r.req.Host.Config == nil {
		return r.NotFoundBuiltIn()
	}

	cnf := r.req.Host.Config
	customNotFound := ErrorFile(cnf)

	if customNotFound == nil {
		return r.NotFoundBuiltIn()
	}

	file, err := r.client.GetFile(integrations.GetFileArgs{
		Location:     cnf.StorageLocation,
		FileName:     customNotFound.FileName,
		DeploymentID: cnf.DeploymentID,
	})

	if err != nil || file == nil {
		return r.NotFoundBuiltIn()
	}

	headers := shttp.HeadersFromMap(customNotFound.Headers)
	headers.Set("Content-Type", file.ContentType)

	r.res = &shttp.Response{
		Status:  http.StatusNotFound,
		Data:    file.Content,
		Headers: headers,
	}

	return r.res
}

func (r *RequestServer) fileContent(headers http.Header) ([]byte, error) {
	shouldOptimize := strings.HasPrefix(headers.Get("Content-Type"), "image") && r.req.Query().Has("size")

	// Check from cache if file exists
	if shouldOptimize {
		if content, _ := r.CachedImage(); content != nil {
			return content, nil
		}
	}

	file, err := r.client.GetFile(integrations.GetFileArgs{
		Location:     r.req.Host.Config.StorageLocation,
		DeploymentID: r.req.Host.Config.DeploymentID,
		FileName:     r.fileMeta.Name,
	})

	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil, nil
	}

	if shouldOptimize {
		optimized, err := r.OptimizeImage(file.Content)

		if err != nil {
			slog.Errorf("error while optimizing image: %s", err.Error())
		}

		if optimized != nil {
			return optimized, nil
		}
	}

	return file.Content, nil
}

// imageKey returns the full path to the current optimized image.
func (r *RequestServer) imageKey() string {
	if r.imgName == "" {
		r.imgName = fmt.Sprintf(
			"%s:%s%s",
			r.req.Host.Config.DeploymentID.String(),
			r.req.Query().Get("size"),
			r.fileMeta.Name,
		)
	}

	return r.imgName
}

func (r *RequestServer) CachedImage() ([]byte, error) {
	image, err := r.cache.Get(r.req.Context(), r.imageKey()).Result()

	if err == redis.Nil {
		return nil, nil
	}

	return []byte(image), err
}

func (r *RequestServer) OptimizeImage(content []byte) ([]byte, error) {
	query := r.req.Query()
	thumb := query.Get("smart")
	size := strings.Split(query.Get("size"), "x")
	width := utils.StringToInt(size[0])
	height := 0

	if len(size) > 1 {
		height = utils.StringToInt(size[1])
	}

	if width == 0 && height == 0 {
		return content, nil
	}

	// Security: do not allow creating images larger than 2048 pixels
	if width > 2048 || height > 2048 {
		return content, nil
	}

	ctx := r.req.Context()
	key := fmt.Sprintf("%d-%s", r.req.Host.Config.DeploymentID, r.fileMeta.Name)
	num, _ := r.cache.Get(ctx, key).Int()

	if num > MAX_IMAGE_VARIANTS {
		slog.Infof("image already has more than 5 variants: %s", r.fileMeta.Name)
		return content, nil
	}

	optimizer := NewImageOptimizer()
	optimized, err := optimizer.Optimize(content, width, height, thumb == "true")

	if optimized != nil {
		if err := r.cache.Set(ctx, key, num+1, time.Hour*24).Err(); err != nil {
			if err != context.Canceled {
				slog.Errorf("error while writing image variant count: %s", err.Error())
			}
		}

		if err := r.cache.Set(ctx, r.imageKey(), optimized, time.Hour*24).Err(); err != nil {
			if err != context.Canceled {
				slog.Errorf("error while writing optimized image: %s", err.Error())
			}
		}
	}

	return optimized, err
}

func shouldInject(_ *RequestContext, res *shttp.Response) bool {
	// We only inject snippets into HTML responses. Encoding is handled by
	// injectSnippets, which decompresses gzip/deflate bodies before injecting.
	if res == nil || !strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html") {
		return false
	}

	return true
}

// responseBytes returns the raw (possibly still-encoded) response body bytes.
func responseBytes(res *shttp.Response) ([]byte, bool) {
	switch data := res.Data.(type) {
	case string:
		return []byte(data), true
	case []byte:
		return data, true
	default:
		return nil, false
	}
}

// decodeBody returns the body as plaintext, transparently decompressing gzip
// and deflate payloads. The bool is false when the encoding is one we can't
// decode (e.g. brotli), so the caller can leave the response untouched rather
// than corrupt it.
func decodeBody(encoding string, data []byte) ([]byte, bool) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "identity":
		return data, true
	case "gzip":
		zr, err := gzip.NewReader(bytes.NewReader(data))

		if err != nil {
			return nil, false
		}

		defer zr.Close()

		out, err := io.ReadAll(zr)

		if err != nil {
			return nil, false
		}

		return out, true
	case "deflate":
		fr := flate.NewReader(bytes.NewReader(data))
		defer fr.Close()

		out, err := io.ReadAll(fr)

		if err != nil {
			return nil, false
		}

		return out, true
	default:
		return nil, false
	}
}

// injectSnippets injects the configured snippets into the HTML response body.
// It transparently decodes gzip/deflate-encoded bodies before injecting, then
// drops the now-stale Content-Encoding and Content-Length headers so the edge
// gzip middleware can re-compress the modified response for the client.
func injectSnippets(req *RequestContext, res *shttp.Response) *shttp.Response {
	if req.Host.Config.Snippets == nil || !shouldInject(req, res) {
		return res
	}

	raw, ok := responseBytes(res)

	if !ok {
		return res
	}

	decoded, ok := decodeBody(res.Headers.Get("Content-Encoding"), raw)

	if !ok {
		// Encoding we can't decode (e.g. brotli); leave the response as-is
		// rather than risk corrupting it.
		return res
	}

	body := string(decoded)

	if body == "" {
		return res
	}

	// We need to use the original path because of path rewrites.
	filters := appconf.SnippetFilters{RequestPath: req.OriginalPath, RequestID: req.RequestID}
	snpt := appconf.SnippetsHTML(req.Host.Config.Snippets, filters)

	injected := body
	injected = insertAfter(injected, "<head>", snpt.HeadPrepend)
	injected = insertAfter(injected, "<body>", snpt.BodyPrepend)
	injected = insertBefore(injected, "</head>", snpt.HeadAppend)
	injected = insertBefore(injected, "</body>", snpt.BodyAppend)

	if injected == body {
		// Nothing matched; keep the original response (and its encoding) as-is.
		return res
	}

	res.Data = injected

	// The body is now plaintext and a different length. Drop the stale framing
	// and encoding; the edge gzip middleware re-compresses from scratch.
	res.Headers.Del("Content-Encoding")
	res.Headers.Del("Content-Length")

	return res
}

func insertBefore(str, pattern, replace string) string {
	if replace == "" {
		return str
	}

	return strings.Replace(str, pattern, replace+pattern, 1)
}

func insertAfter(str, tag, text string) string {
	if text == "" {
		return str
	}

	tagStartIndex := strings.Index(str, tag[:len(tag)-1])

	if tagStartIndex == -1 {
		return str
	}

	tagEndIndex := strings.Index(str[tagStartIndex:], ">")

	if tagEndIndex == -1 {
		return str
	}

	index := tagStartIndex + tagEndIndex + 1
	return str[:index] + text + str[index:]
}

func injectHeaders(req *RequestContext, res *shttp.Response) *shttp.Response {
	if res == nil {
		return nil
	}

	if res.Headers == nil {
		res.Headers = make(http.Header)
	}

	res.Headers.Set("x-sk-version", req.Host.Config.DeploymentID.String())

	if !stormkitServerHeaderOff {
		res.Headers.Set("Server", "Stormkit")
	}

	if req.Host.IsStormkitSubdomain && res.Headers.Get("x-robots-tag") == "" {
		res.Headers.Set("x-robots-tag", "noindex")
	}

	// If we're here, it probably means that the dynamic request returned no content.
	if res.Headers.Get("content-type") == "" {
		res.Headers.Set("content-type", "text/html; charset=utf-8")
	}

	// Apply user-configured custom headers last so they take precedence over
	// any defaults set above. Matches against the request URL path so the
	// same rules cover static, dynamic, SSR and proxied responses. An empty
	// value (e.g. the `!Header-Name` negate syntax in _headers) removes the
	// header rather than emitting it with an empty value.
	if len(req.Host.Config.CustomHeaders) > 0 {
		overrides := deploy.ApplyHeaders(strings.ToLower(req.URL().Path), nil, req.Host.Config.CustomHeaders)

		for k, v := range overrides {
			if v == "" {
				res.Headers.Del(k)
			} else {
				res.Headers.Set(k, v)
			}
		}
	}

	return res
}

func analyticsRecord(req *RequestContext, res *shttp.Response) *analytics.Record {
	if req.Host == nil || req.Host.Config == nil {
		return nil
	}

	// Stormkit's reserved endpoints (auth pages, the analytics beacon, etc.) are
	// platform infrastructure, not visitor page views — some render text/html and
	// would otherwise inflate the customer's analytics.
	if strings.HasPrefix(req.URL().Path, reservedPathPrefix) {
		return nil
	}

	if req.Host.Config.DomainID == 0 && !config.IsDevelopment() {
		return nil
	}

	// Do not count XHR requests and ignore records non-html records.
	if strings.EqualFold(req.Header.Get("X-Requested-With"), "xmlhttprequest") {
		return nil
	}

	userAgent := req.UserAgent()

	if analytics.IsBot(userAgent) {
		return nil
	}

	if !analytics.IsUtf8(userAgent) {
		return nil
	}

	referrer := analytics.NormalizeReferrer(req.Referer())

	return &analytics.Record{
		AppID:       req.Host.Config.AppID,
		EnvID:       req.Host.Config.EnvID,
		VisitorIP:   req.RemoteIP(),
		RequestTS:   utils.NewUnix(),
		RequestPath: req.OriginalPath,
		StatusCode:  res.Status,
		Referrer:    null.NewString(referrer, referrer != ""),
		UserAgent:   null.NewString(userAgent, userAgent != ""),
		DomainID:    req.Host.Config.DomainID,
		RequestID:   null.NewString(req.RequestID, req.RequestID != ""),
		Source:      null.StringFrom("server"),
	}
}

// ErrorFile returns the first static file that is configured as an error page.
// It checks the configured error file, and if not found, it falls back to
// the default error files (404.html, 500.html, error.html).
func ErrorFile(cnf *appconf.Config) *appconf.StaticFile {
	lookup := []string{
		cnf.ErrorFile,
		"/404.html",
		"/500.html",
		"/error.html",
	}

	for _, v := range lookup {
		if v == "" {
			continue
		}

		if file := cnf.StaticFiles[v]; file != nil {
			return file
		}
	}

	return nil
}

// headersSize calculates the approximate memory size of HTTP headers.
func headersSize(m map[string][]string) int64 {
	var size int64

	// Iterate through all keys and values
	for key, values := range m {
		size = size + int64(len(key))

		// Size of each string in the slice
		for _, val := range values {
			size += int64(len(val))
		}
	}

	return size
}
