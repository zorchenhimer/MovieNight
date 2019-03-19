package common

// Misc utils

import (
	"fmt"
	"math/rand"
	"regexp"
)

var re_username *regexp.Regexp = regexp.MustCompile(`^[0-9a-zA-Z_-]+$`)

func IsValidName(name string) bool {
	return re_username.MatchString(name)
}

func RandomColor() string {
	nums := []int32{}
	for i := 0; i < 6; i++ {
		nums = append(nums, rand.Int31n(15))
	}
	return fmt.Sprintf("#%X%X%X%X%X%X",
		nums[0], nums[1], nums[2],
		nums[3], nums[4], nums[5])
}
