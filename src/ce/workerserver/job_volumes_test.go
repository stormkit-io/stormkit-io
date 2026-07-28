package jobs

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type JobVolumesSuite struct {
	suite.Suite
	*factory.Factory
	conn   databasetest.TestDB
	tmpdir string
}

func (s *JobVolumesSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	tmpdir, err := os.MkdirTemp("", "tmp-stale-volumes-")
	s.NoError(err)
	s.tmpdir = tmpdir

	s.NoError(admin.Store().UpsertConfig(context.Background(), admin.InstanceConfig{
		VolumesConfig: &admin.VolumesConfig{
			MountType: volumes.FileSys,
			RootPath:  tmpdir,
		},
	}))
}

func (s *JobVolumesSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	os.RemoveAll(s.tmpdir)
}

func (s *JobVolumesSuite) writeFile(envID types.ID, name string) *volumes.File {
	content := []byte("Hello world!")

	file := &volumes.File{
		EnvID:     envID,
		Name:      name,
		Size:      int64(len(content)),
		Path:      s.tmpdir,
		IsPublic:  true,
		CreatedAt: utils.NewUnix(),
	}

	s.NoError(os.WriteFile(file.FullPath(), content, 0664))
	s.NoError(volumes.Store().Insert(context.Background(), []*volumes.File{file}, envID))

	return file
}

func (s *JobVolumesSuite) countVolumes(envID types.ID) int {
	var count int

	s.NoError(s.conn.
		QueryRow("SELECT COUNT(*) FROM volumes WHERE env_id = $1", envID).
		Scan(&count),
	)

	return count
}

func (s *JobVolumesSuite) Test_RemoveStaleVolumes() {
	app := s.MockApp(nil)

	deletedEnv := s.MockEnv(app, map[string]any{
		"DeletedAt": utils.Unix{Time: time.Now(), Valid: true},
	})

	liveEnv := s.MockEnv(app)

	staleFile := s.writeFile(deletedEnv.ID, "stale.txt")
	liveFile := s.writeFile(liveEnv.ID, "live.txt")

	s.NoError(removeStaleVolumes(context.Background(), NewStore()))

	s.Equal(0, s.countVolumes(deletedEnv.ID), "volume rows for the deleted env should be removed")
	s.NoFileExists(staleFile.FullPath(), "physical file for the deleted env should be removed")

	s.Equal(1, s.countVolumes(liveEnv.ID), "volume rows for the live env must be preserved")
	s.FileExists(liveFile.FullPath(), "physical file for the live env must be preserved")
}

func (s *JobVolumesSuite) Test_RemoveStaleVolumes_NoVolumesConfig() {
	s.NoError(admin.Store().UpsertConfig(context.Background(), admin.InstanceConfig{}))

	app := s.MockApp(nil)

	deletedEnv := s.MockEnv(app, map[string]any{
		"DeletedAt": utils.Unix{Time: time.Now(), Valid: true},
	})

	s.writeFile(deletedEnv.ID, "stale.txt")

	// Without a configured volume backend the physical bytes cannot be removed,
	// so the rows must be left intact for a later run rather than orphaning files.
	s.NoError(removeStaleVolumes(context.Background(), NewStore()))
	s.Equal(1, s.countVolumes(deletedEnv.ID))
}

func TestJobVolumesSuite(t *testing.T) {
	suite.Run(t, &JobVolumesSuite{})
}
