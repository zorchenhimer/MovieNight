package common

// Misc utils

import (
	"regexp"
)

var usernameRegex *regexp.Regexp = regexp.MustCompile(`^[0-9a-zA-Z_-]+$`)

// IsValidName checks that name is within the correct ranges, follows the regex defined
// and is not a valid color name
func IsValidName(name string) bool {
	return 3 <= len(name) && len(name) <= 36 &&
		usernameRegex.MatchString(name)
}
