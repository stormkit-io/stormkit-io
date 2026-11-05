package integrations

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// SignedUploadsTTL is how long links stay valid. The runner asks for them right
// before it uses them, so they only need to outlast one upload; storage checks
// the expiry when a request starts, not when it ends.
const SignedUploadsTTL = 10 * time.Minute

// signedRequestAttempts is how many times a request through a link is tried
// before the upload or download fails.
const signedRequestAttempts = 4

// signedRetryDelay is the wait before the first retry; it doubles after each.
var signedRetryDelay = time.Second

// artifactNames are the zips a build can upload, as the bundler names them.
var artifactNames = []string{"sk-client.zip", "sk-server.zip", "sk-api.zip", "sk-migrations.zip"}

// defaultMaxArtifactSize caps each deployment zip on self-hosted instances
// unless they set config.MaxArtifactSizeEnvVar.
const defaultMaxArtifactSize = 1 << 30 // 1GiB

// CloudArtifactSizeLimits are the largest deployment zips Stormkit Cloud
// accepts. Zips without an entry get the self-hosted default.
var CloudArtifactSizeLimits = map[string]int64{
	"sk-client.zip": 50<<20 + 1024,  // 50MB
	"sk-server.zip": 100<<20 + 1024, // 100MB
	"sk-api.zip":    100<<20 + 1024, // 100MB
}

var signedUploadClient = &http.Client{Timeout: 15 * time.Minute}

// UploadSigner is implemented by storage clients that can give a build signed
// links instead of storage credentials.
type UploadSigner interface {
	// SignsUploads reports whether the client signs links. Clients that
	// cannot do without credentials, such as one publishing to Lambda, do not.
	SignsUploads() bool

	// SignUploads returns the upload forms of the deployment's zips. It
	// returns nil when the client does not sign links.
	SignUploads(context.Context, SignUploadsParams) (*SignedUploads, error)

	// VerifyUploads returns an error when a location in the upload result a
	// build reported is not one of the deployment's own zips. The links limit
	// what a build can write, not what it reports, and the result decides
	// what the deployment serves. It does nothing for clients that do not
	// sign links.
	VerifyUploads(VerifyUploadsParams) error
}

// VerifyUploadsParams names the deployment and the upload result its build
// reported.
type VerifyUploadsParams struct {
	AppID        types.ID
	EnvID        types.ID
	DeploymentID types.ID
	Result       *UploadResult
}

// SignedPost is a signed form that uploads one object. Storage rejects files
// larger than MaxSize.
type SignedPost struct {
	URL     string            `json:"url"`
	Fields  map[string]string `json:"fields"`
	MaxSize int64             `json:"maxSize"`
}

// SignedArtifact is the upload form of one deployment zip and the location it
// is stored at once uploaded.
type SignedArtifact struct {
	SignedPost
	Location string `json:"location"`
}

// SignedUploads are short-lived forms that let a build upload its
// deployment's artifacts without storage credentials. Each form is valid for
// one object only.
type SignedUploads struct {
	Artifacts map[string]SignedArtifact `json:"artifacts"` // Zip name, such as sk-client.zip, to its form
}

// SignUploadsParams names the deployment to sign forms for.
type SignUploadsParams struct {
	AppID        types.ID
	DeploymentID types.ID
}

// SignsUploads implements UploadSigner. Builds that publish functions to
// Lambda need credentials, so no links are signed for them.
func (a *AWSClient) SignsUploads() bool {
	return a.lambdaClient == nil
}

// SignUploads implements UploadSigner.
func (a *AWSClient) SignUploads(ctx context.Context, p SignUploadsParams) (*SignedUploads, error) {
	if !a.SignsUploads() {
		return nil, nil
	}

	return a.signUploads(ctx, signUploadsParams{bucket: a.defaultBucket(), deployment: p})
}

// VerifyUploads implements UploadSigner.
func (a *AWSClient) VerifyUploads(p VerifyUploadsParams) error {
	if !a.SignsUploads() {
		return nil
	}

	return a.verifyUploads(verifyUploadsParams{bucket: a.defaultBucket(), deployment: p})
}

// SignsUploads implements UploadSigner.
func (c *HCloudClient) SignsUploads() bool {
	return true
}

// SignUploads implements UploadSigner.
func (c *HCloudClient) SignUploads(ctx context.Context, p SignUploadsParams) (*SignedUploads, error) {
	return c.s3.signUploads(ctx, signUploadsParams{bucket: c.bucket, deployment: p})
}

// VerifyUploads implements UploadSigner.
func (c *HCloudClient) VerifyUploads(p VerifyUploadsParams) error {
	return c.s3.verifyUploads(verifyUploadsParams{bucket: c.bucket, deployment: p})
}

func (a *AWSClient) defaultBucket() string {
	if conf := config.Get().AWS; conf != nil {
		return conf.StorageBucket
	}

	return ""
}

// artifactKey returns the key a deployment zip is stored at.
func artifactKey(appID, deploymentID types.ID, name string) string {
	return fmt.Sprintf("%d/%d/%s", appID, deploymentID, name)
}

type signUploadsParams struct {
	bucket     string
	deployment SignUploadsParams
}

func (a *AWSClient) signUploads(ctx context.Context, p signUploadsParams) (*SignedUploads, error) {
	if p.bucket == "" {
		return nil, errors.New("no storage bucket is configured")
	}

	d := p.deployment
	presigner := s3.NewPresignClient(a.S3Client, s3.WithPresignExpires(SignedUploadsTTL))

	uploads := &SignedUploads{Artifacts: map[string]SignedArtifact{}}

	for _, name := range artifactNames {
		key := artifactKey(d.AppID, d.DeploymentID, name)
		form, err := signPost(ctx, signPostParams{presigner: presigner, bucket: p.bucket, key: key, maxSize: maxArtifactSize(name)})

		if err != nil {
			return nil, err
		}

		uploads.Artifacts[name] = SignedArtifact{
			SignedPost: form,
			Location:   fmt.Sprintf("%s:%s/%s", a.providerPrefix, p.bucket, key),
		}
	}

	return uploads, nil
}

// maxArtifactSize returns the size limit of a deployment zip.
func maxArtifactSize(name string) int64 {
	if limit, ok := CloudArtifactSizeLimits[name]; ok && config.IsStormkitCloud() {
		return limit
	}

	if mb := config.MaxArtifactSizeMB(); mb > 0 {
		return mb << 20
	}

	return defaultMaxArtifactSize
}

type signPostParams struct {
	presigner *s3.PresignClient
	bucket    string
	key       string
	maxSize   int64
}

// signPost signs a form that uploads one object of at most maxSize bytes.
// Unlike a signed PUT, the form makes storage enforce the size.
func signPost(ctx context.Context, p signPostParams) (SignedPost, error) {
	req, err := p.presigner.PresignPostObject(ctx, &s3.PutObjectInput{Bucket: &p.bucket, Key: &p.key}, func(o *s3.PresignPostOptions) {
		o.Expires = SignedUploadsTTL
		o.Conditions = []any{[]any{"content-length-range", 0, p.maxSize}}
	})

	if err != nil {
		return SignedPost{}, err
	}

	return SignedPost{URL: req.URL, Fields: req.Values, MaxSize: p.maxSize}, nil
}

type verifyUploadsParams struct {
	bucket     string
	deployment VerifyUploadsParams
}

func (a *AWSClient) verifyUploads(p verifyUploadsParams) error {
	d := p.deployment

	if d.Result == nil {
		return nil
	}

	prefix := fmt.Sprintf("%s:%s/%s", a.providerPrefix, p.bucket, artifactKey(d.AppID, d.DeploymentID, ""))

	for _, overview := range []UploadOverview{d.Result.Client, d.Result.Server, d.Result.API, d.Result.Migrations} {
		if overview.Location == "" {
			continue
		}

		key, ok := strings.CutPrefix(overview.Location, prefix)

		// Function locations point at the handler inside the zip.
		name, _, _ := strings.Cut(key, "/")

		if !ok || strings.Contains(key, "..") || !slices.Contains(artifactNames, name) {
			return fmt.Errorf("%s is not an upload of this deployment", overview.Location)
		}
	}

	return nil
}

// Upload uploads the deployment zips through their links.
func (u *SignedUploads) Upload(args UploadArgs) (*UploadResult, error) {
	var err error

	result := &UploadResult{}

	if result.Client, err = u.upload(signedUploadParams{zip: args.ClientZip}); err != nil {
		return nil, err
	}

	if result.Server, err = u.upload(signedUploadParams{zip: args.ServerZip, handler: args.ServerHandler}); err != nil {
		return nil, err
	}

	if result.API, err = u.upload(signedUploadParams{zip: args.APIZip, handler: args.APIHandler}); err != nil {
		return nil, err
	}

	if result.Migrations, err = u.upload(signedUploadParams{zip: args.MigrationsZip}); err != nil {
		return nil, err
	}

	return result, nil
}

type signedUploadParams struct {
	zip     string
	handler string
}

// upload uploads one zip and returns where it is stored. Function locations
// point at the handler inside the zip.
func (u *SignedUploads) upload(p signedUploadParams) (UploadOverview, error) {
	if p.zip == "" {
		return UploadOverview{}, nil
	}

	artifact, ok := u.Artifacts[path.Base(p.zip)]

	if !ok {
		return UploadOverview{}, fmt.Errorf("no upload link for %s", path.Base(p.zip))
	}

	params := signedPostParams{form: artifact.SignedPost, file: p.zip}

	if !config.IsStormkitCloud() {
		params.hint = fmt.Sprintf(" (set %s on the instance to change it)", config.MaxArtifactSizeEnvVar)
	}

	size, err := u.post(params)

	if err != nil {
		return UploadOverview{}, fmt.Errorf("cannot upload %s: %w", path.Base(p.zip), err)
	}

	location := artifact.Location

	if p.handler != "" {
		location = location + "/" + p.handler
	}

	return UploadOverview{
		FilesUploaded: 1,
		BytesUploaded: size,
		Location:      location,
	}, nil
}

type signedPostParams struct {
	form SignedPost
	file string
	hint string // How to raise the limit, for the error of a file over it
}

// post uploads a file through a signed form and returns its size.
func (u *SignedUploads) post(p signedPostParams) (int64, error) {
	ctx := context.Background()
	f, err := os.Open(p.file)

	if err != nil {
		return 0, err
	}

	defer f.Close()

	stat, err := f.Stat()

	if err != nil {
		return 0, err
	}

	// Storage rejects it as well; checking first names the limit.
	if p.form.MaxSize > 0 && stat.Size() > p.form.MaxSize {
		return 0, fmt.Errorf("the file is %dMB while the limit is %dMB%s", stat.Size()>>20, p.form.MaxSize>>20, p.hint)
	}

	// The form fields come first and the file last, as storage requires. They
	// are written once so the body can be sent again with a known length.
	var head bytes.Buffer

	w := multipart.NewWriter(&head)

	for k, v := range p.form.Fields {
		if err := w.WriteField(k, v); err != nil {
			return 0, err
		}
	}

	if _, err := w.CreateFormFile("file", path.Base(p.file)); err != nil {
		return 0, err
	}

	tail := "\r\n--" + w.Boundary() + "--\r\n"

	res, err := u.do(ctx, func() (*http.Request, error) {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}

		body := io.MultiReader(bytes.NewReader(head.Bytes()), f, strings.NewReader(tail))
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.form.URL, body)

		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", w.FormDataContentType())
		req.ContentLength = int64(head.Len()) + stat.Size() + int64(len(tail))

		return req, nil
	})

	if err != nil {
		return 0, err
	}

	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))

		return 0, fmt.Errorf("upload returned %s: %s", res.Status, body)
	}

	return stat.Size(), nil
}

// do sends the request newRequest builds, retrying connection errors and
// server errors with a growing delay. Storage answers those for brief outages
// and throttling, which the SDK retries for credential-based uploads too.
func (u *SignedUploads) do(ctx context.Context, newRequest func() (*http.Request, error)) (*http.Response, error) {
	delay := signedRetryDelay

	for attempt := 1; ; attempt++ {
		req, err := newRequest()

		if err != nil {
			return nil, err
		}

		res, err := signedUploadClient.Do(req)

		retry := err != nil || res.StatusCode >= http.StatusInternalServerError || res.StatusCode == http.StatusTooManyRequests

		if !retry || attempt == signedRequestAttempts || ctx.Err() != nil {
			return res, err
		}

		if res != nil {
			res.Body.Close()
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}

		delay *= 2
	}
}
