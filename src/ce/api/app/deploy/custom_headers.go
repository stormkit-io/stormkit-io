package deploy

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
)

type CustomHeader struct {
	Location string
	Key      string
	Value    string
	re       *regexp.Regexp
}

// ApplyHeaders adds the matching `headers` to `current` if the header location
// matches with the file name. If duplicate headers are found, `headers` wins.
func ApplyHeaders(fileName string, current HeaderKeyValue, headers []CustomHeader) HeaderKeyValue {
	if current == nil {
		current = make(HeaderKeyValue)
	}

	for _, header := range headers {
		if header.re != nil && header.re.MatchString(fileName) {
			current[header.Key] = header.Value
		}
	}

	return current
}

// ParseHeaders will parse the headers file and return custom headers.
func ParseHeaders(customHeaders string) ([]CustomHeader, error) {
	headers := []CustomHeader{}
	location := "/*"
	var locationRegexp *regexp.Regexp
	var err error

	// See https://docs.netlify.com/routing/headers/#syntax-for-the-headers-file for
	// more information on the syntax. We do respect Netlify's syntax to allow easier
	// migration.
	for _, line := range strings.Split(customHeaders, "\n") {
		line = strings.TrimSpace(line)

		// It's a comment, skip.
		if strings.HasPrefix(line, "#") {
			continue
		}

		// It's a file location, so set for further requests.
		if strings.HasPrefix(line, "/") {
			location = line

			pattern := strings.Replace(line, "*", "(.*)", -1)
			pattern = "^" + pattern + "$"
			locationRegexp, err = regexp.Compile(pattern)

			if err != nil {
				return nil, err
			}

			continue
		}

		// Negate
		// edit 2025-09-04: not sure why this is needed but
		// leaving to make sure we don't break anything
		if strings.HasPrefix(line, "!") {
			headers = append(headers, CustomHeader{
				re:       locationRegexp,
				Location: location,
				Key:      strings.TrimSpace(strings.Replace(line, "!", "", 1)),
				Value:    "",
			})

			continue
		}

		pieces := strings.SplitN(line, ":", 2)

		if len(pieces) == 2 {
			headers = append(headers, CustomHeader{
				re:       locationRegexp,
				Location: location,
				Key:      strings.TrimSpace(pieces[0]),
				Value:    strings.TrimSpace(pieces[1]),
			})
		} else if line != "" {
			return nil, fmt.Errorf("invalid syntax (missing colon): %s", line)
		}
	}

	return headers, nil
}

// ParseHeadersFile will parse the headers file and return custom headers.
func ParseHeadersFile(headersFile string) ([]CustomHeader, error) {
	if headersFile == "" {
		return nil, nil
	}

	if !file.Exists(headersFile) {
		return nil, nil
	}

	data, err := os.ReadFile(headersFile)

	if err != nil || data == nil {
		return nil, err
	}

	return ParseHeaders(string(data))
}
