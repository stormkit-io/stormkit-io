package sysstats

import (
	"strconv"
	"strings"
)

// parseInfoUint pulls a numeric field out of a Redis INFO response, whose lines
// are "key:value" separated by CRLF.
func parseInfoUint(info, field string) uint64 {
	prefix := field + ":"

	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)

		if !strings.HasPrefix(line, prefix) {
			continue
		}

		value, err := strconv.ParseUint(strings.TrimPrefix(line, prefix), 10, 64)

		if err != nil {
			return 0
		}

		return value
	}

	return 0
}
