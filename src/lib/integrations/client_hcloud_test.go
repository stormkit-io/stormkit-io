package integrations_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go/middleware"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

// fakeObjectStorage is the part of the S3 API the HCloud client uses, served
// with path-style addressing.
type fakeObjectStorage struct {
	mu      sync.Mutex
	objects map[string][]byte      // <bucket>/<key>
	headers map[string]http.Header // request headers of each PUT
}

func newFakeObjectStorage() *fakeObjectStorage {
	return &fakeObjectStorage{
		objects: map[string][]byte{},
		headers: map[string]http.Header{},
	}
}

func (f *fakeObjectStorage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	bucket, key, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/"), "/")
	query := r.URL.Query()

	switch {
	case r.Method == http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		f.objects[bucket+"/"+key] = body
		f.headers[bucket+"/"+key] = r.Header.Clone()
		w.Header().Set("ETag", `"etag"`)

	case r.Method == http.MethodPost && query.Has("delete"):
		req := struct {
			Objects []struct {
				Key string `xml:"Key"`
			} `xml:"Object"`
		}{}

		_ = xml.NewDecoder(r.Body).Decode(&req)

		for _, o := range req.Objects {
			delete(f.objects, bucket+"/"+o.Key)
		}

		fmt.Fprint(w, `<DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></DeleteResult>`)

	case r.Method == http.MethodGet && query.Get("list-type") == "2":
		keys := []string{}

		for k := range f.objects {
			if rest, ok := strings.CutPrefix(k, bucket+"/"); ok && strings.HasPrefix(rest, query.Get("prefix")) {
				keys = append(keys, rest)
			}
		}

		sort.Strings(keys)

		var b strings.Builder

		for _, k := range keys {
			fmt.Fprintf(&b, "<Contents><Key>%s</Key></Contents>", k)
		}

		fmt.Fprintf(w, `<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>%s</Name><KeyCount>%d</KeyCount><IsTruncated>false</IsTruncated>%s</ListBucketResult>`, bucket, len(keys), b.String())

	case r.Method == http.MethodGet:
		body, ok := f.objects[bucket+"/"+key]

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
			return
		}

		start, end := 0, len(body)-1

		if rng, ok := strings.CutPrefix(r.Header.Get("Range"), "bytes="); ok {
			from, to, _ := strings.Cut(rng, "-")
			start, _ = strconv.Atoi(from)

			if n, err := strconv.Atoi(to); err == nil && n < end {
				end = n
			}

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(body)))
			w.Header().Set("Content-Length", strconv.Itoa(end-start+1))
			w.WriteHeader(http.StatusPartialContent)
		}

		_, _ = w.Write(body[start : end+1])

	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

// has reports whether the object exists.
func (f *fakeObjectStorage) has(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	_, ok := f.objects[key]

	return ok
}

// remove deletes an object behind the client's back.
func (f *fakeObjectStorage) remove(key string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.objects, key)
}

type HCloudSuite struct {
	suite.Suite

	tmpdir     string
	storage    *fakeObjectStorage
	server     *httptest.Server
	storageDir string
}

func (s *HCloudSuite) BeforeTest(_, _ string) {
	tmpdir, err := os.MkdirTemp("", "hcloud-")
	s.Require().NoError(err)

	s.tmpdir = tmpdir
	s.storage = newFakeObjectStorage()
	s.server = httptest.NewServer(s.storage)
	s.storageDir = config.Get().Deployer.StorageDir

	config.Get().Deployer.StorageDir = path.Join(tmpdir, "deployments")
	config.Get().HCloud = &config.HCloudConfig{
		AccessKey:     "access-key",
		SecretKey:     "secret-key",
		StorageBucket: "artifacts",
		Region:        "fsn1",
		Endpoint:      s.server.URL,
	}

	s.zip("sk-client.zip", map[string]string{"index.html": "Hello world"})
	s.zip("sk-server.zip", map[string]string{
		"server.js": `exports.handler = (req, ctx, cb) => cb(null, { status: 200, body: "rendered " + req.path })`,
	})
	s.zip("sk-migrations.zip", map[string]string{"001.sql": "select 1;"})
}

func (s *HCloudSuite) AfterTest(_, _ string) {
	s.server.Close()

	config.Get().HCloud = nil
	config.Get().AWS = nil
	config.Get().Deployer.StorageDir = s.storageDir
	integrations.CachedHCloudClient = nil
	integrations.CachedAWSClient = nil

	os.RemoveAll(s.tmpdir)
}

// zip writes a zip with the given files into the temp dir.
func (s *HCloudSuite) zip(name string, files map[string]string) {
	f, err := os.Create(path.Join(s.tmpdir, name))
	s.Require().NoError(err)

	defer f.Close()

	w := zip.NewWriter(f)

	for fileName, content := range files {
		entry, err := w.Create(fileName)
		s.Require().NoError(err)

		_, err = entry.Write([]byte(content))
		s.Require().NoError(err)
	}

	s.Require().NoError(w.Close())
}

func (s *HCloudSuite) client() *integrations.HCloudClient {
	client, err := integrations.HCloud(integrations.ClientArgs{})
	s.Require().NoError(err)

	return client
}

// upload uploads the suite's zips as deployment 7 of app 3.
func (s *HCloudSuite) upload() *integrations.UploadResult {
	result, err := s.client().Upload(integrations.UploadArgs{
		AppID:         3,
		DeploymentID:  7,
		ClientZip:     path.Join(s.tmpdir, "sk-client.zip"),
		ServerZip:     path.Join(s.tmpdir, "sk-server.zip"),
		ServerHandler: "server.js:handler",
		MigrationsZip: path.Join(s.tmpdir, "sk-migrations.zip"),
	})

	s.Require().NoError(err)

	return result
}

func (s *HCloudSuite) Test_Upload() {
	result := s.upload()

	s.Equal("hcloud:artifacts/3/7/sk-client.zip", result.Client.Location)
	s.Equal("hcloud:artifacts/3/7/sk-server.zip/server.js:handler", result.Server.Location)
	s.Equal("hcloud:artifacts/3/7/sk-migrations.zip", result.Migrations.Location)
	s.Empty(result.API.Location)

	s.True(s.storage.has("artifacts/3/7/sk-client.zip"))
	s.True(s.storage.has("artifacts/3/7/sk-server.zip"))
	s.True(s.storage.has("artifacts/3/7/sk-migrations.zip"))
}

// Test_Upload_NoEncryptionHeader verifies that uploads leave out the SSE-S3
// header, which Hetzner does not document support for.
func (s *HCloudSuite) Test_Upload_NoEncryptionHeader() {
	s.upload()

	s.Empty(s.storage.headers["artifacts/3/7/sk-client.zip"].Get("X-Amz-Server-Side-Encryption"))
}

// Test_GetFile_ServesFromDisk verifies that a deployment is downloaded once
// and then served from the local copy.
func (s *HCloudSuite) Test_GetFile_ServesFromDisk() {
	location := s.upload().Client.Location

	file, err := s.client().GetFile(integrations.GetFileArgs{Location: location, FileName: "index.html"})
	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal("Hello world", string(file.Content))

	s.storage.remove("artifacts/3/7/sk-client.zip")

	file, err = s.client().GetFile(integrations.GetFileArgs{Location: location, FileName: "index.html"})
	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal("Hello world", string(file.Content))
}

// Test_GetFile_ServesFromAnotherServer verifies that a server that did not
// build the deployment fetches it from the bucket.
func (s *HCloudSuite) Test_GetFile_ServesFromAnotherServer() {
	location := s.upload().Client.Location

	integrations.CachedHCloudClient = nil
	integrations.CachedAWSClient = nil
	config.Get().Deployer.StorageDir = path.Join(s.tmpdir, "other-server")

	file, err := s.client().GetFile(integrations.GetFileArgs{Location: location, FileName: "index.html"})
	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal("Hello world", string(file.Content))
}

func (s *HCloudSuite) Test_GetFile_MissingFile() {
	location := s.upload().Client.Location

	file, err := s.client().GetFile(integrations.GetFileArgs{Location: location, FileName: "missing.html"})
	s.NoError(err)
	s.Nil(file)
}

func (s *HCloudSuite) Test_GetFile_MissingDeployment() {
	file, err := s.client().GetFile(integrations.GetFileArgs{
		Location: "hcloud:artifacts/3/8/sk-client.zip",
		FileName: "index.html",
	})

	s.NoError(err)
	s.Nil(file)
}

// Test_GetFile_Migrations verifies that a location naming only a zip returns
// the zip itself.
func (s *HCloudSuite) Test_GetFile_Migrations() {
	location := s.upload().Migrations.Location
	want, err := os.ReadFile(path.Join(s.tmpdir, "sk-migrations.zip"))
	s.Require().NoError(err)

	file, err := s.client().GetFile(integrations.GetFileArgs{Location: location})
	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal(want, file.Content)
}

func (s *HCloudSuite) Test_GetFile_InvalidLocation() {
	_, err := s.client().GetFile(integrations.GetFileArgs{Location: "hcloud:artifacts", FileName: "index.html"})
	s.Error(err)
}

// Test_RejectsPathTraversal verifies that a location cannot point outside the
// local storage directory, whether it is read, run or deleted.
func (s *HCloudSuite) Test_RejectsPathTraversal() {
	victim := path.Join(s.tmpdir, "victim")
	s.Require().NoError(os.MkdirAll(victim, 0774))

	locations := []string{
		"hcloud:../../victim.zip",
		"hcloud:artifacts/../../../victim.zip",
		"hcloud:artifacts/3/7/../../victim.zip",
		"hcloud:artifacts/3/7/sk-server.zip/../../../server.js:handler",
		"hcloud:/etc/3/7/sk-client.zip",
		"hcloud:artifacts/app/7/sk-client.zip",
	}

	for _, location := range locations {
		_, err := s.client().GetFile(integrations.GetFileArgs{Location: location, FileName: "index.html"})
		s.Error(err, location)

		_, err = s.client().Invoke(integrations.InvokeArgs{ARN: location, URL: &url.URL{}})
		s.Error(err, location)

		err = s.client().DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{StorageLocation: location})
		s.Error(err, location)
	}

	s.DirExists(victim)
}

func (s *HCloudSuite) Test_Invoke() {
	location := s.upload().Server.Location

	result, err := s.client().Invoke(integrations.InvokeArgs{
		ARN:    location,
		Method: shttp.MethodGet,
		URL:    &url.URL{Path: "/about"},
	})

	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal(http.StatusOK, result.StatusCode)
	s.Equal("rendered /about", string(result.Body))
}

// Test_Invoke_ServerCmd verifies that a long-running server can be found
// under its stored location, which is how hosting stops a deployment that
// failed its warm-up.
func (s *HCloudSuite) Test_Invoke_ServerCmd() {
	s.zip("sk-server.zip", map[string]string{
		"server.js": `require("http").createServer((req, res) => res.end("served")).listen(process.env.PORT, "127.0.0.1")`,
	})

	location := s.upload().Server.Location
	pm := integrations.Filesys().ProcessManager()

	defer pm.KillAll()

	args := integrations.InvokeArgs{
		ARN:          location,
		Method:       shttp.MethodGet,
		URL:          &url.URL{},
		Command:      "node server.js",
		DeploymentID: 7,
	}

	var result *integrations.InvokeResult
	var err error

	for i := 0; i < 60; i++ {
		if result, err = s.client().Invoke(args); err != nil || result.Headers.Get("Retry-After") == "" {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal("served", string(result.Body))

	service := pm.GetService(location)
	s.Require().NotNil(service)

	service.Kill()
	s.Nil(pm.GetService(location))
}

func (s *HCloudSuite) Test_DeleteArtifacts() {
	result := s.upload()
	client := s.client()

	file, err := client.GetFile(integrations.GetFileArgs{Location: result.Client.Location, FileName: "index.html"})
	s.Require().NoError(err)
	s.Require().NotNil(file)

	s.NoError(client.DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{
		StorageLocation:  result.Client.Location,
		FunctionLocation: result.Server.Location,
	}))

	s.False(s.storage.has("artifacts/3/7/sk-client.zip"))
	s.False(s.storage.has("artifacts/3/7/sk-server.zip"))
	s.False(s.storage.has("artifacts/3/7/sk-migrations.zip"))
	s.NoDirExists(path.Join(s.tmpdir, "deployments", "hcloud", "artifacts", "3", "7"))

	file, err = client.GetFile(integrations.GetFileArgs{Location: result.Client.Location, FileName: "index.html"})
	s.NoError(err)
	s.Nil(file)
}

// Test_DeleteArtifacts_KeepsOtherDeploymentsCached verifies that deleting
// deployment 7 leaves the cached files of deployment 70 alone.
func (s *HCloudSuite) Test_DeleteArtifacts_KeepsOtherDeploymentsCached() {
	client := s.client()
	deleted := s.upload().Client.Location

	kept, err := client.Upload(integrations.UploadArgs{
		AppID:        3,
		DeploymentID: 70,
		ClientZip:    path.Join(s.tmpdir, "sk-client.zip"),
	})

	s.Require().NoError(err)

	for _, location := range []string{deleted, kept.Client.Location} {
		file, err := client.GetFile(integrations.GetFileArgs{Location: location, FileName: "index.html"})
		s.Require().NoError(err)
		s.Require().NotNil(file)
	}

	s.NoError(client.DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{StorageLocation: deleted}))

	// With the file gone from the disk, only memory can still serve it.
	s.Require().NoError(os.Remove(path.Join(s.tmpdir, "deployments", "hcloud", "artifacts", "3", "70", "sk-client", "index.html")))

	file, err := client.GetFile(integrations.GetFileArgs{Location: kept.Client.Location, FileName: "index.html"})
	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal("Hello world", string(file.Content))
}

func (s *HCloudSuite) Test_CacheArtifacts() {
	archive := path.Join(s.tmpdir, "cache.tar.gz")
	s.Require().NoError(os.WriteFile(archive, []byte("cache"), 0664))

	args := integrations.CacheArtifactArgs{AppID: 3, EnvID: 5, Dir: "node_modules", LocalPath: archive}
	s.NoError(s.client().UploadCacheArtifact(context.Background(), args))

	args.LocalPath = path.Join(s.tmpdir, "restored.tar.gz")
	found, err := s.client().DownloadCacheArtifact(context.Background(), args)
	s.NoError(err)
	s.True(found)

	content, err := os.ReadFile(args.LocalPath)
	s.NoError(err)
	s.Equal("cache", string(content))
}

func (s *HCloudSuite) Test_MissingCredentials() {
	config.Get().HCloud.AccessKey = ""

	_, err := integrations.HCloud(integrations.ClientArgs{})
	s.Error(err)
}

// awsClient sets up the shared AWS client so that every GetObject returns
// body, as if the object were in an S3 bucket.
func (s *HCloudSuite) awsClient(body string) *integrations.AWSClient {
	config.Get().AWS = &config.AwsConfig{Region: "eu-central-1", StorageBucket: "aws-bucket"}

	client, err := integrations.AWS(integrations.ClientArgs{
		AccessKey: "aws-access-key",
		SecretKey: "aws-secret-key",
		Middlewares: []func(stack *middleware.Stack) error{
			func(stack *middleware.Stack) error {
				return stack.Finalize.Add(
					middleware.FinalizeMiddlewareFunc("GetObject", func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
						if awsmiddleware.GetOperationName(ctx) != "GetObject" {
							return next.HandleFinalize(ctx, in)
						}

						return middleware.FinalizeOutput{
							Result: &s3.GetObjectOutput{
								Body:          io.NopCloser(bytes.NewReader([]byte(body))),
								ContentLength: utils.Ptr(int64(len(body))),
							},
						}, middleware.Metadata{}, nil
					}),
					middleware.Before,
				)
			},
		},
	}, nil)

	s.Require().NoError(err)

	return client
}

// localDeployment writes deployment 1 the way the filesystem provider stores
// it and returns its directory.
func (s *HCloudSuite) localDeployment() string {
	dir := path.Join(s.tmpdir, "local", "deployment-1")

	s.Require().NoError(os.MkdirAll(path.Join(dir, "client"), 0774))
	s.Require().NoError(os.MkdirAll(path.Join(dir, "server"), 0774))
	s.Require().NoError(os.WriteFile(path.Join(dir, "client", "index.html"), []byte("Local world"), 0664))
	s.Require().NoError(os.WriteFile(path.Join(dir, "server", "server.js"), []byte(
		`exports.handler = (req, ctx, cb) => cb(null, { status: 200, body: "local " + req.path })`,
	), 0664))

	return dir
}

// Test_Upload_KeepsAWSClient verifies that HCloud does not take over the
// shared AWS client, so both can be used in the same process.
func (s *HCloudSuite) Test_Upload_KeepsAWSClient() {
	awsClient := s.awsClient("from aws")

	result := s.upload()

	s.Equal("hcloud:artifacts/3/7/sk-client.zip", result.Client.Location)
	s.True(s.storage.has("artifacts/3/7/sk-client.zip"))
	s.Same(awsClient, integrations.CachedAWSClient)
	s.Equal("AWS", integrations.CachedAWSClient.Name())

	file, err := awsClient.GetFile(integrations.GetFileArgs{Location: "aws:aws-bucket/1/2/robots.txt"})
	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal("from aws", string(file.Content))
}

// Test_GetFile_LegacyAWS verifies that deployments stored on S3 before the
// switch are still served from there.
func (s *HCloudSuite) Test_GetFile_LegacyAWS() {
	s.awsClient("from aws")

	file, err := s.client().GetFile(integrations.GetFileArgs{Location: "aws:aws-bucket/1/2/robots.txt"})
	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal("from aws", string(file.Content))
}

func (s *HCloudSuite) Test_GetFile_LegacyAWS_NotConfigured() {
	_, err := s.client().GetFile(integrations.GetFileArgs{Location: "aws:aws-bucket/1/2/robots.txt"})
	s.Error(err)
}

// Test_GetFile_LegacyLocal verifies that deployments stored on the local disk
// before the switch are still served.
func (s *HCloudSuite) Test_GetFile_LegacyLocal() {
	dir := s.localDeployment()

	file, err := s.client().GetFile(integrations.GetFileArgs{
		Location: "local:" + path.Join(dir, "client"),
		FileName: "index.html",
	})

	s.NoError(err)
	s.Require().NotNil(file)
	s.Equal("Local world", string(file.Content))
}

func (s *HCloudSuite) Test_Invoke_LegacyLocal() {
	dir := s.localDeployment()

	result, err := s.client().Invoke(integrations.InvokeArgs{
		ARN:    "local:" + path.Join(dir, "server", "server.js") + ":handler",
		Method: shttp.MethodGet,
		URL:    &url.URL{Path: "/home"},
	})

	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal("local /home", string(result.Body))
}

func (s *HCloudSuite) Test_DeleteArtifacts_LegacyLocal() {
	dir := s.localDeployment()

	s.NoError(s.client().DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{
		StorageLocation:  "local:" + path.Join(dir, "client"),
		FunctionLocation: "local:" + path.Join(dir, "server", "server.js") + ":handler",
	}))

	s.NoDirExists(dir)
}

func TestHCloudSuite(t *testing.T) {
	suite.Run(t, new(HCloudSuite))
}
