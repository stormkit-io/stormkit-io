package authwallhandlers_test

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

// teamFixture is a team with an owner, a developer and two apps.
type teamFixture struct {
	s         *suite.Suite
	team      team.Team
	owner     *factory.MockUser
	developer *factory.MockUser
	app       *factory.MockApp
	otherApp  *factory.MockApp
	env       *factory.MockEnv
}

func newTeamFixture(s *suite.Suite, f *factory.Factory) *teamFixture {
	tf := &teamFixture{s: s, owner: f.MockUser(), developer: f.MockUser(), team: team.Team{Name: "Acme"}}
	store := team.NewStore()

	s.Require().NoError(store.CreateTeam(context.Background(), &tf.team, &team.Member{
		UserID: tf.owner.ID,
		Role:   team.ROLE_OWNER,
		Status: true,
	}))

	s.Require().NoError(store.AddMemberToTeam(context.Background(), &team.Member{
		TeamID: tf.team.ID,
		UserID: tf.developer.ID,
		Role:   team.ROLE_DEVELOPER,
		Status: true,
	}))

	tf.app = f.MockApp(tf.owner, map[string]any{"TeamID": tf.team.ID})
	tf.otherApp = f.MockApp(tf.owner, map[string]any{"TeamID": tf.team.ID})
	tf.env = f.MockEnv(tf.app)

	return tf
}

// login creates a login in the team that opens app.
func (tf *teamFixture) login(email string, appIDs ...types.ID) *authwall.AuthWall {
	if len(appIDs) == 0 {
		appIDs = []types.ID{tf.app.ID}
	}

	aw := &authwall.AuthWall{
		TeamID:        tf.team.ID,
		LoginEmail:    email,
		LoginPassword: "123pass-word",
		AppIDs:        appIDs,
	}

	tf.s.Require().NoError(authwall.Store().CreateLogin(context.Background(), aw))

	return aw
}

// request sends a request to the Auth Wall services as usr.
func (tf *teamFixture) request(usr *factory.MockUser, method, path string, body any) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		method,
		path,
		body,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)
}

// opens reports whether the credentials open the wall of the environment.
func (tf *teamFixture) opens(envID types.ID, email, password string) bool {
	aw, err := authwall.Store().Login(context.Background(), authwall.LoginParams{
		EnvID:    envID,
		Email:    email,
		Password: password,
	})

	tf.s.Require().NoError(err)

	return aw != nil
}

// lastAudit returns the most recent audit entry of the team.
func (tf *teamFixture) lastAudit() *audit.Audit {
	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{TeamID: tf.team.ID})
	tf.s.Require().NoError(err)

	if len(audits) == 0 {
		return nil
	}

	return &audits[0]
}
