package deployhandlers_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type DeployStartTestSuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	mockDeployer *mocks.Deployer
	mockUploader mocks.RunnerUploaderInterface
}

func (s *DeployStartTestSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployer = &mocks.Deployer{}
	s.mockUploader = mocks.RunnerUploaderInterface{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	runner.DefaultUploader = &s.mockUploader
	deployservice.MockDeployer = s.mockDeployer
}

func (s *DeployStartTestSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	runner.DefaultUploader = nil
	deployservice.MockDeployer = nil
}

func (s *DeployStartTestSuite) createZipWithFiles() []byte {
	// Create a buffer to hold the zip content
	var buf bytes.Buffer

	// Create a new zip writer
	zipWriter := zip.NewWriter(&buf)

	files := map[string][]byte{
		"index.html":     []byte("Hello World"),
		"style.css":      []byte("body { background: #000; }"),
		"script.js":      []byte("console.log('Hello World');"),
		"redirects.json": []byte(`[{ "from": "stormkit.io", "to": "www.stormkit.io" }]`),
	}

	for fileName, fileContent := range files {
		zf, err := zipWriter.Create(fileName)
		s.NoError(err)
		_, err = zf.Write(fileContent)
		s.NoError(err)
	}

	// Close the zip writer to finalize the zip content
	s.NoError(zipWriter.Close())

	// Return the zip content as a byte slice
	return buf.Bytes()
}

func (s *DeployStartTestSuite) Test_Success_Github() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl, map[string]any{
		"AutoPublish": true,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy",
		map[string]any{
			"appId":   appl.ID.String(),
			"envId":   env.ID.String(),
			"branch":  "master",
			"publish": false,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	d := &deploy.Deployment{}
	s.Nil(json.Unmarshal([]byte(response.String()), d))
	s.Equal(response.Code, http.StatusOK)
	s.Equal(env.ID, d.EnvID)
	s.Equal(d.Branch, "master")

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything, mock.MatchedBy(func(_app *app.App) bool {
			return s.Equal(appl.ID, _app.ID)
		}),
		mock.MatchedBy(func(_depl *deploy.Deployment) bool {
			return s.Equal(_depl.CheckoutRepo, "github/svedova/react-minimal") &&
				s.False(_depl.ShouldPublish)
		}),
	)
}

func (s *DeployStartTestSuite) Test_Success_Bitbucket() {
	usr := s.MockUser()
	appl := s.MockApp(usr, map[string]any{
		"Repo": "bitbucket/stormkit-io/app-stormkit-io",
	})

	env := s.MockEnv(appl, map[string]any{
		"AutoPublish": true,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy",
		map[string]any{
			"appId":   appl.ID.String(),
			"envId":   env.ID.String(),
			"branch":  "main",
			"publish": true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	d := &deploy.Deployment{}
	s.Nil(json.Unmarshal([]byte(response.String()), d))
	s.Equal(response.Code, http.StatusOK)
	s.Equal(env.ID, d.EnvID)
	s.Equal("main", d.Branch)

	s.mockDeployer.AssertCalled(s.T(), "Deploy",
		mock.Anything, mock.MatchedBy(func(_app *app.App) bool {
			return s.Equal(appl.ID, _app.ID)
		}),
		mock.MatchedBy(func(_depl *deploy.Deployment) bool {
			return s.Equal(_depl.CheckoutRepo, "bitbucket/stormkit-io/app-stormkit-io") &&
				s.True(_depl.ShouldPublish) &&
				s.Equal("main", _depl.Branch) &&
				s.Equal("npm run build", _depl.BuildConfig.BuildCmd) &&
				s.Equal("build", _depl.BuildConfig.DistFolder) &&
				s.Equal(map[string]string{
					"NODE_ENV": "production",
				}, _depl.BuildConfig.Vars)
		}),
	)
}

func (s *DeployStartTestSuite) Test_Zip() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	existingDeployment := s.MockDeployment(env)

	s.mockUploader.On("Upload", mock.Anything).Return(&integrations.UploadResult{
		Client: integrations.UploadOverview{
			BytesUploaded: 2042,
			FilesUploaded: 1,
			Location:      "local:/my/path/sk-client.zip",
		},
	}, nil)

	requestBody, contentType, err := shttptest.MultipartForm(map[string][]byte{
		"appId": []byte(app.ID.String()),
		"envId": []byte(env.ID.String()),
	}, map[string][]shttptest.UploadFile{
		"files": {
			{Name: "my.zip", Data: string(s.createZipWithFiles())},
		},
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy",
		requestBody,
		map[string]string{
			"Content-Type":  contentType,
			"Authorization": usertest.Authorization(usr.ID),
			"X-File-ID":     "my-zip",
		},
	)

	did := existingDeployment.ID + 1
	myDeployment, err := deploy.NewStore().DeploymentByID(context.Background(), did)

	s.NoError(err)
	s.NotNil(myDeployment)

	tmpl := template.Must(template.New("batchInsert").Parse(`{
	  "id": "{{ .id }}",
	  "appId": "{{ .appId }}",
	  "envId": "{{ .envId}}",
	  "envName": "production",
	  "duration": {{ .duration }},
	  "apiPathPrefix": "",
	  "uploadResult": {
	  	"clientBytes": 2042,
		"serverlessBytes": 0,
		"serverBytes": 0
	  },
	  "displayName": "{{ .displayName }}",
	  "error": "",
	  "repo": "",
	  "snapshot": null,
	  "stoppedManually": false,
	  "isAutoDeploy": false,
	  "isAutoPublish": false,
	  "published": [],
	  "previewUrl": "http://{{ .displayName }}--{{ .id }}.stormkit:8888",
	  "detailsUrl": "/apps/{{ .appId }}/environments/{{ .envId }}/deployments/{{ .id }}",
	  "statusChecksPassed": null,
	  "statusChecks": [],
	  "createdAt": "{{ .createdAt }}",
	  "stoppedAt": "{{ .stoppedAt }}",
	  "status": "success",
	  "commit": {
		"author": "David Lorenzo",
		"message": "",
		"sha": ""
	  },
	  "branch": "",
	  "logs": [
		{ "duration": 0, "message": "\nSuccessfully deployed client side.\nTotal bytes uploaded: 2.0kB\n\n", "payload": null, "status": true, "title": "deploy" } 
	  ]
	}`))

	var wr bytes.Buffer
	s.NoError(tmpl.Execute(&wr, map[string]any{
		"id":          myDeployment.ID,
		"appId":       app.ID,
		"envId":       env.ID,
		"displayName": app.DisplayName,
		"duration":    myDeployment.StoppedAt.Unix() - myDeployment.CreatedAt.Unix(),
		"createdAt":   myDeployment.CreatedAt.UnixStr(),
		"stoppedAt":   myDeployment.StoppedAt.UnixStr(),
	}))

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(wr.String(), response.String())
	s.Equal(myDeployment.BuildManifest.Redirects, []redirects.Redirect{{From: "stormkit.io", To: "www.stormkit.io"}})

	// Should not auto publish
	depls, err := deploy.NewStore().MyDeployments(context.Background(), &deploy.DeploymentsQueryFilters{
		EnvID:     env.ID,
		Published: utils.Ptr(true),
	})

	s.NoError(err)
	s.Len(depls, 0)
}

func (s *DeployStartTestSuite) Test_Zip_AutoPublish() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.mockUploader.On("Upload", mock.Anything).Return(&integrations.UploadResult{
		Client: integrations.UploadOverview{
			BytesUploaded: 2042,
			FilesUploaded: 1,
			Location:      "local:/my/path/sk-client.zip",
		},
	}, nil)

	requestBody, contentType, err := shttptest.MultipartForm(map[string][]byte{
		"appId":   []byte(app.ID.String()),
		"envId":   []byte(env.ID.String()),
		"publish": []byte("true"),
	}, map[string][]shttptest.UploadFile{
		"files": {
			{Name: "my.zip", Data: string(s.createZipWithFiles())},
		},
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy",
		requestBody,
		map[string]string{
			"Content-Type":  contentType,
			"Authorization": usertest.Authorization(usr.ID),
			"X-File-ID":     "my-zip-2",
		},
	)

	depls, err := deploy.NewStore().MyDeployments(context.Background(), &deploy.DeploymentsQueryFilters{
		EnvID:     env.ID,
		Published: utils.Ptr(true),
	})

	s.NoError(err)
	s.Len(depls, 1)

	s.Equal(http.StatusOK, response.Code)
}

func (s *DeployStartTestSuite) Test_BadRequest() {
	app := s.GetApp()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/deploy",
		map[string]any{
			"appId": app.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(1),
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerDeployStart(t *testing.T) {
	suite.Run(t, &DeployStartTestSuite{})
}
