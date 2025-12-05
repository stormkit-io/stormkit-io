package hosting

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

const MAX_IMAGE_VARIANTS = 5
const SESSION_COOKIE_NAME = "stormkit_session"

var stormkitServerHeaderOff = os.Getenv("STORMKIT_SERVER_HEADER") == "off"

// HandlerForward forwards all requests
func HandlerForward(req *RequestContext) *shttp.Response {
	rs := NewRequestServer(req)

	// Send artifacts such as analytics record, logs, etc to redis queue
	defer func() {
		go rs.artifacts()
	}()

	if rs.req.Host == nil || rs.req.Host.Config == nil {
		return rs.NotFound()
	}

	middlewares := []func(req *RequestContext) (*shttp.Response, error){
		WithAuthWall,
		WithRedirect,
	}

	for _, md := range middlewares {
		res, err := md(req)

		if err != nil {
			return rs.Error(err)
		}

		if res != nil {
			// We still want to be able to serve custom 404 pages
			// when the proxy request returns 404.
			if res.Status == http.StatusNotFound {
				return rs.NotFound()
			}

			return res
		}
	}

	return rs.Handle()
}

type FileMeta struct {
	Name    string
	Headers map[string]string
}

type RequestServer struct {
	req       *RequestContext
	res       *shttp.Response
	client    integrations.ClientInterface
	cache     *redis.Client
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

func (r *RequestServer) artifacts() {
	var data []byte

	if r.res == nil || r.req == nil || r.req.Host == nil || r.req.Host.Config == nil {
		return
	}

	if r.res.Data != nil {
		data, _ = r.res.Data.([]byte)
	}

	Queue(&jobs.HostingRecord{
		AppID:           r.req.Host.Config.AppID,
		EnvID:           r.req.Host.Config.EnvID,
		DeploymentID:    r.req.Host.Config.DeploymentID,
		HostName:        r.req.Host.Name,
		BillingUserID:   r.req.Host.Config.BillingUserID,
		FunctionInvoked: r.fnInvoked,
		Logs:            r.logs,
		Analytics:       r.record,
		TotalBandwidth:  int64(len(data)) + headersSize(r.res.Headers),
	})
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
	defer func() {
		r.res = injectHeaders(r.req, r.res)
		r.res = injectSnippets(r.req, r.res)

		if r.req.Host.Config.IsEnterprise {
			contentType := strings.ToLower(r.res.Headers.Get("Content-Type"))

			if strings.HasPrefix(contentType, "text/html") {
				r.record = analyticsRecord(r.req, r.res)
			}
		}
	}()

	if r.fileMeta = r.FileMeta(); r.fileMeta != nil {
		return r.Static()
	}

	return r.Dynamic()

}

func (r *RequestServer) Static() *shttp.Response {
	notModified := false
	headers := shttp.HeadersFromMap(r.fileMeta.Headers)
	modifiedSinceHeader := r.req.Header.Get("If-Modified-Since")

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

	result, err := integrations.Client().Invoke(integrations.InvokeArgs{
		URL:          url,
		ARN:          arn,
		Body:         r.req.Body,
		Method:       r.req.Method,
		Headers:      r.req.Headers(),
		HostName:     r.req.Host.Name,
		AppID:        cnf.AppID,
		EnvID:        cnf.EnvID,
		DeploymentID: cnf.DeploymentID,
		Command:      cnf.ServerCmd,
		EnvVariables: cnf.EnvVariables,
		IsPublished:  cnf.Percentage > 0,
		CaptureLogs:  true,
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
	// We only need to inject the snippets to the html files.
	// We also skip if the `Content-Encoding` header is given because
	// we're not going to unzip and re-zip.
	if !strings.HasPrefix(res.Headers.Get("Content-Type"), "text/html") ||
		res == nil ||
		res.Headers.Get("Content-Encoding") != "" {
		return false
	}

	return true
}

func responseBody(res *shttp.Response) string {
	switch res.Data.(type) {
	case string:
		return res.Data.(string)
	case []byte:
		return string(res.Data.([]byte))
	default:
		return ""
	}
}

// injectSnippets injects the snippets to the response data.
func injectSnippets(req *RequestContext, res *shttp.Response) *shttp.Response {
	if req.Host.Config.Snippets == nil || !shouldInject(req, res) {
		return res
	}

	// We need to use the original path because of path rewrites.
	filters := appconf.SnippetFilters{RequestPath: req.OriginalPath}
	snpt := appconf.SnippetsHTML(req.Host.Config.Snippets, filters)
	body := responseBody(res)

	if body != "" {
		body = insertAfter(body, "<head>", snpt.HeadPrepend)
		body = insertAfter(body, "<body>", snpt.BodyPrepend)
		body = insertBefore(body, "</head>", snpt.HeadAppend)
		body = insertBefore(body, "</body>", snpt.BodyAppend)
		res.Data = body
	}

	return res
}

func insertBefore(str, pattern, replace string) string {
	if replace == "" {
		return str
	}

	return strings.Replace(str, pattern, replace+"</body>", 1)
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

	return res
}

func analyticsRecord(req *RequestContext, res *shttp.Response) *analytics.Record {
	if req.Host == nil || req.Host.Config == nil {
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
		VisitorIP:   shttp.ClientIP(req.Request), // Proxy-aware IP extraction
		RequestTS:   utils.NewUnix(),
		RequestPath: req.OriginalPath,
		StatusCode:  res.Status,
		Referrer:    null.NewString(referrer, referrer != ""),
		UserAgent:   null.NewString(userAgent, userAgent != ""),
		DomainID:    req.Host.Config.DomainID,
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
