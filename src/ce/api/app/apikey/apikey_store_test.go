package apikey_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type StoreSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
	app  *factory.MockApp
	env  *factory.MockEnv
}

func (s *StoreSuite) SetupSuite() {
	s.conn = databasetest.InitTx("apikey_store_suite")
	s.Factory = factory.New(s.conn)

	s.app = s.MockApp(nil)
	s.env = s.MockEnv(s.app)
}

func (s *StoreSuite) TearDownSuite() {
	s.conn.CloseTx()
}

func (s *StoreSuite) TestAddAPIKey_Success() {
	key := &apikey.Token{
		Name:  "Something",
		Scope: apikey.SCOPE_ENV,
		AppID: s.app.ID,
		EnvID: s.env.ID,
	}

	err := apikey.NewStore().AddAPIKey(context.Background(), key)

	s.NoError(err)
	s.True(key.ID > 0)
}

func (s *StoreSuite) TestAddAPIKey_InvalidScope() {
	key := &apikey.Token{
		Name:  "Something",
		Scope: "something-else",
	}

	err := apikey.NewStore().AddAPIKey(context.Background(), key)

	s.Error(err)
	s.Equal("scope is invalid", err.Error())
}

func (s *StoreSuite) Test_APIKey_ByID() {
	keys := make([]*apikey.Token, 0, 5)

	for i := range 5 {
		key := &apikey.Token{
			Name:  "Something - " + strconv.Itoa(i),
			Scope: apikey.SCOPE_ENV,
			AppID: s.app.ID,
			EnvID: s.env.ID,
			Value: utils.RandomToken(128),
		}

		err := apikey.NewStore().AddAPIKey(context.Background(), key)
		s.NoError(err)
		keys = append(keys, key)
	}

	fetchedKey, err := apikey.NewStore().APIKeyByID(context.Background(), keys[3].ID)
	s.NoError(err)
	s.Equal(keys[3].ID, fetchedKey.ID)
	s.Equal(keys[3].Name, fetchedKey.Name)
	s.Equal(keys[3].Scope, fetchedKey.Scope)
	s.Equal(keys[3].AppID, fetchedKey.AppID)
	s.Equal(keys[3].EnvID, fetchedKey.EnvID)
}

func (s *StoreSuite) Test_APIKey_ByToken() {
	tkn := "SK_" + utils.RandomToken(128)
	key := &apikey.Token{
		Name:  "API Key",
		Scope: apikey.SCOPE_ENV,
		AppID: s.app.ID,
		EnvID: s.env.ID,
		Value: tkn,
	}

	err := apikey.NewStore().AddAPIKey(context.Background(), key)
	s.NoError(err)

	// Legacy approach with raw value
	fetchedKey, err := apikey.NewStore().APIKey(context.Background(), tkn)
	s.NoError(err)
	s.Equal(key.ID, fetchedKey.ID)
	s.Equal(key.Name, fetchedKey.Name)
	s.Equal(key.Scope, fetchedKey.Scope)
	s.Equal(key.AppID, fetchedKey.AppID)
	s.Equal(key.EnvID, fetchedKey.EnvID)

	// Make sure it's not possible to fetch a key by it's hashed value
	fetchedKey, err = apikey.NewStore().APIKey(context.Background(), utils.SHA256Hash([]byte(tkn)))
	s.NoError(err)
	s.Nil(fetchedKey)
}

func TestStoreSuite(t *testing.T) {
	suite.Run(t, &StoreSuite{})
}
