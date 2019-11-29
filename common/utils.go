package common

// Misc utils

import (
	"regexp"
)

var usernameRegex *regexp.Regexp = regexp.MustCompile(`^[0-9a-zA-Z_-]*[a-zA-Z0-9]+[0-9a-zA-Z_-]*$`)

const InvalidNameError string = `Invalid name.<br />Name must be between 3 and 36 characters in length; contain only numbers, letters, underscores or dashes; and contain at least one number or letter.<br />Names cannot contain spaces.`

// IsValidName checks that name is within the correct ranges, follows the regex defined
// and is not a valid color name
func IsValidName(name string) bool {
	return 3 <= len(name) && len(name) <= 36 &&
		usernameRegex.MatchString(name)
}
