package common

// Misc utils

import (
	"fmt"
	"math/rand"
	"regexp"
)

var usernameRegex *regexp.Regexp = regexp.MustCompile(`^[0-9a-zA-Z_-]+$`)

// IsValidName checks that name is within the correct ranges, follows the regex defined
// and is not a valid color name
func IsValidName(name string) bool {
	return 3 <= len(name) && len(name) <= 36 &&
		usernameRegex.MatchString(name) && !IsValidColor(name)
}

// RandomColor returns a hex color code
func RandomColor() string {
	nums := []int32{}
	for i := 0; i < 6; i++ {
		nums = append(nums, rand.Int31n(15))
	}
	return fmt.Sprintf("#%X%X%X%X%X%X",
		nums[0], nums[1], nums[2],
		nums[3], nums[4], nums[5])
}
