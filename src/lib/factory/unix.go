package factory

import (
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var OriginalNewUnix = utils.NewUnix

// MockNewUnix is used for tests
func MockNewUnix() utils.Unix {
	t, err := time.ParseInLocation(time.DateTime, "2024-04-06 15:45:30", time.UTC)

	if err != nil {
		panic(err)
	}

	return utils.Unix{Valid: true, Time: t.UTC()}
}
