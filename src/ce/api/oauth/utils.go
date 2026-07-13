package oauth

import "strings"

// ParseRepo parses the given repository and returns an owner, name (repository name) pair.
func ParseRepo(repo string) (owner, name string) {
	if strings.HasPrefix(repo, "github/") {
		pieces := strings.Split(repo[7:], "/")
		return pieces[0], strings.Join(pieces[1:], "/")
	} else if strings.HasPrefix(repo, "bitbucket/") {
		pieces := strings.Split(repo[10:], "/")
		return pieces[0], strings.Join(pieces[1:], "/")
	} else if strings.HasPrefix(repo, "gitlab") {
		pieces := strings.Split(repo[7:], "/")
		return pieces[0], strings.Join(pieces[1:], "/")
	} else {
		pieces := strings.Split(repo, "/")
		return pieces[0], strings.Join(pieces[1:], "/")
	}
}
