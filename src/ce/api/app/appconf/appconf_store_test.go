package appconf_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stretchr/testify/suite"
)

type AppConfStoreSuite struct {
	suite.Suite
}

func (s *AppConfStoreSuite) Test_NormalizeHeaders() {
	type HeaderTest struct {
		FileName        string
		ActualHeaders   map[string]string
		ExpectedHeaders map[string]string
	}

	tests := []HeaderTest{
		{
			FileName:      "/subfolder/index.html",
			ActualHeaders: map[string]string{},
			ExpectedHeaders: map[string]string{
				"content-type": "text/html; charset=utf-8",
			},
		},
		{
			FileName:      "/subfolder/index.js",
			ActualHeaders: map[string]string{},
			ExpectedHeaders: map[string]string{
				"content-type": "application/javascript; charset=utf-8",
			},
		},
		{
			FileName:      "/subfolder/index.css",
			ActualHeaders: map[string]string{},
			ExpectedHeaders: map[string]string{
				"content-type": "text/css; charset=utf-8",
			},
		},
		{
			FileName:      "/subfolder/favicon.ico",
			ActualHeaders: map[string]string{},
			ExpectedHeaders: map[string]string{
				"content-type": "image/x-icon",
			},
		},
		{
			FileName: "/subfolder/favicon.png",
			ActualHeaders: map[string]string{
				"content-type": "custom-type",
			},
			ExpectedHeaders: map[string]string{
				"content-type": "custom-type",
			},
		},
	}

	for _, test := range tests {
		file := appconf.StaticFile{
			FileName: test.FileName,
			Headers:  appconf.NormalizeHeaders(test.FileName, test.ActualHeaders),
		}

		// This will be added dynamically
		file.Headers["content-type"] = test.ExpectedHeaders["content-type"]

		s.Equal(file, appconf.StaticFile{
			FileName: test.FileName,
			Headers:  test.ExpectedHeaders,
		})
	}
}

func TestAppConfStoreSui(t *testing.T) {
	suite.Run(t, new(AppConfStoreSuite))
}
