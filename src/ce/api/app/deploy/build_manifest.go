package deploy

import (
	"crypto/sha1"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
)

var ignoredExtensions = []string{
	"js.map",
	"css.map",
}

// These are path prefixes that we ignore when calculating the etag.
// Make sure these paths are lowercase.
var ignoredPathPrefixes = []string{
	"/__macosx",
}

// This is an additional header map for TypeByExtension.
var AdditionalMimeTypesLower = map[string]string{
	".aac":   "audio/aac",
	".avi":   "video/x-msvideo",
	".bmp":   "image/bmp",
	".csv":   "text/csv; charset=utf-8",
	".doc":   "application/msword",
	".docx":  "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".eot":   "application/vnd.ms-fontobject",
	".flac":  "audio/flac",
	".ico":   "image/x-icon",
	".mp3":   "audio/mpeg",
	".mp4":   "video/mp4",
	".ogg":   "audio/ogg",
	".otf":   "font/otf",
	".rar":   "application/x-rar-compressed",
	".rtf":   "application/rtf",
	".tar":   "application/x-tar",
	".tiff":  "image/tiff",
	".ts":    "video/mp2t",
	".ttf":   "font/ttf",
	".txt":   "text/plain; charset=utf-8",
	".wav":   "audio/wav",
	".webm":  "video/webm",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".xhtml": "application/xhtml+xml",
	".xls":   "application/vnd.ms-excel",
	".xlsx":  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".zip":   "application/zip",
}

type Redirect = redirects.Redirect

type CDNFile struct {
	Name    string            `json:"fileName"`
	Headers map[string]string `json:"headers,omitempty"`
}

type APIFile struct {
	FileName     string `json:"fileName"`
	SetupFile    bool   `json:"setupFile,omitempty"`
	TeardownFile bool   `json:"teardownFile,omitempty"`
}

type HeaderKeyValue = map[string]string
type StaticFiles = map[string]HeaderKeyValue

type BuildManifest struct {
	Success         bool                 `json:"-"`
	Redirects       []redirects.Redirect `json:"redirects,omitempty"`
	APIRoutes       []string             `json:"apiRoutes,omitempty"`
	Runtimes        []string             `json:"runtimes,omitempty"`        // Runtimes to install
	Headers         []CustomHeader       `json:"headers,omitempty"`         // Headers
	StaticFiles     StaticFiles          `json:"staticFiles,omitempty"`     // FileName => Headers
	CDNFiles        []CDNFile            `json:"cdnFiles"`                  // @deprecated: use StaticFiles instead
	APIFiles        []APIFile            `json:"apiFiles,omitempty"`        // @deprecated: use APIRoutes instead
	FunctionHandler string               `json:"functionHandler,omitempty"` // file_name.js:handler_name
	APIHandler      string               `json:"apiHandler,omitempty"`      // file_name.js:handler_name
}

// Scan implements the Scanner interface.
func (bm *BuildManifest) Scan(value any) error {
	if value != nil {
		if b, ok := value.([]byte); ok {
			if err := json.Unmarshal(b, bm); err != nil {
				return err
			}
		}
	}

	return nil
}

// Value implements the Sql Driver interface.
func (bm *BuildManifest) Value() (driver.Value, error) {
	if bm == nil {
		return nil, nil
	}

	if len(bm.CDNFiles) == 0 &&
		len(bm.APIFiles) == 0 &&
		len(bm.Redirects) == 0 &&
		len(bm.StaticFiles) == 0 &&
		len(bm.APIRoutes) == 0 &&
		bm.FunctionHandler == "" {
		return nil, nil
	}

	return json.Marshal(bm)
}

// PrepareStaticFiles returns a list of files with the etag header.
// This is used to include in the manifest.
func PrepareStaticFiles(dirs []string, headers []CustomHeader) StaticFiles {
	files := map[string]map[string]string{}

	if len(dirs) == 0 {
		return nil
	}

	for _, dir := range dirs {
		if !file.Exists(dir) {
			continue
		}

		_ = filepath.WalkDir(dir, func(pathToFile string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			for _, ext := range ignoredExtensions {
				if strings.HasSuffix(pathToFile, ext) {
					return nil
				}
			}

			fileName := strings.Replace(pathToFile, dir, "", 1)

			for _, path := range ignoredPathPrefixes {
				if strings.HasPrefix(strings.ToLower(fileName), path) {
					return nil
				}
			}

			if pathToFile == dir || info.IsDir() || fileName == "" {
				return nil
			}

			files[fileName] = map[string]string{
				"etag":         CalculateETag(pathToFile, false),
				"content-type": CalculateContentType(pathToFile),
			}

			for _, header := range headers {
				pattern := strings.Replace(header.Location, "*", "(.*)", -1)
				matched, _ := regexp.MatchString(pattern, fileName)

				if matched {
					files[fileName][header.Key] = header.Value
				}
			}

			return nil
		})
	}

	return files
}

// CalculateETag calculates the etag for the given file.
func CalculateETag(filePath string, weak bool) string {
	body, err := os.ReadFile(filePath)

	if err != nil {
		return ""
	}

	hash := sha1.Sum(body)
	etag := fmt.Sprintf("\"%d-%x\"", int(len(hash)), hash)

	if weak {
		etag = "W/" + etag
	}

	return etag
}

// CalculateContentType determines the content type for the given file.
func CalculateContentType(pathToFile string) string {
	ext := path.Ext(pathToFile)

	// Use built-in package to find the content-type
	if ext != "" {
		if contentType := mime.TypeByExtension(ext); contentType != "" {
			return contentType
		}
	}

	// If it's not found, use the additional mime types
	if contentType := AdditionalMimeTypesLower[ext]; contentType != "" {
		return contentType
	}

	// If it's still not found, default to `text/plain`
	return "text/plain"
}

// ParseRedirects will parse the given files and return redirects.
// This function will also Netlify style _redirects.
func ParseRedirects(redirectsFiles []string) ([]redirects.Redirect, error) {
	for _, redirectsFile := range redirectsFiles {
		if !file.Exists(redirectsFile) {
			continue
		}

		data, err := os.ReadFile(redirectsFile)

		if err != nil {
			return nil, err
		}

		if strings.HasSuffix(redirectsFile, ".json") {
			redirects := []redirects.Redirect{}
			err := json.Unmarshal(data, &redirects)
			return redirects, err
		}

		return parseNetlifyRedirects(redirectsFile)
	}

	return nil, nil
}

func parseNetlifyRedirects(redirectsFile string) ([]redirects.Redirect, error) {
	doc, err := os.ReadFile(redirectsFile)

	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(doc), "\n")
	reds := []redirects.Redirect{}

	for _, line := range lines {
		pieces := strings.Fields(line)

		// Invalid statement, ignore it.
		if len(pieces) < 2 {
			continue
		}

		redirect := redirects.Redirect{
			From: pieces[0],
			To:   strings.Replace(pieces[1], ":splat", "$1", 1),
		}

		if len(pieces) > 2 {
			redirect.Status, _ = strconv.Atoi(strings.ReplaceAll(pieces[2], "!", ""))
		}

		// Special case, make sure it's not a hard redirect.
		if strings.Contains(redirect.From, "*") && strings.HasSuffix(redirect.To, ".html") {
			redirect.Assets = false
			redirect.Status = 0
		} else if redirect.Status > 0 && string(strconv.Itoa(redirect.Status)[0]) != "3" {
			redirect.Status = 0
		} else if redirect.Status == 0 {
			redirect.Status = http.StatusMovedPermanently
		}

		reds = append(reds, redirect)
	}

	return reds, nil
}
