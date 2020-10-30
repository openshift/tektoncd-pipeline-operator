package exclusion

import "regexp"

// IsNamespaceExcluded determines if the namespace is in
// the denylist.
func IsNamespaceExcluded(exclusionList []string, namespace string) bool {
	for _, n := range exclusionList {
		isMatched, _ := regexp.MatchString(n, namespace)
		if isMatched || n == namespace {
			return true
		}
	}
	return false
}
