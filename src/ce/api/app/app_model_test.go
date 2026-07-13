package app_test

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type AppModelSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *AppModelSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *AppModelSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *AppModelSuite) Test_Secret() {
	a := app.App{ID: 1}
	secret := a.Secret()
	decoded, _ := utils.DecodeString(secret)
	decrypt, _ := utils.Decrypt(decoded)
	s.Equal(int(binary.LittleEndian.Uint32(decrypt)), 1)
}

func (s *AppModelSuite) Test_Validate_Valid() {
	app := app.New(1)
	app.Repo = "bitbucket/stormkit-io/stormkit-io"
	s.Nil(app.Validate())
}

func (s *AppModelSuite) Test_Validate_InvalidRepoProvider() {
	a := app.New(1)
	a.Repo = "gitlab.com/stormkit-io/my-repo.git"
	err := a.Validate()

	s.Error(err)
	s.Equal(err.Errors["repo"], app.ErrRepoInvalidProvider.Error())
	s.Equal(err.Errors["domain"], "")
}

func (s *AppModelSuite) Test_Cache_PrivateKey() {
	a := app.New(1)
	pk := a.PrivateKey(context.Background())
	s.Equal(pk, a.PrivateKey(context.Background()))
}

func TestAppModel(t *testing.T) {
	suite.Run(t, &AppModelSuite{})
}
