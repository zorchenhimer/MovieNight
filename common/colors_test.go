package common

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHexThreeToSix(t *testing.T) {
	result, err := hexThreeToSix("RGB")

	assert.NoError(t, err)
	assert.Equal(t, "RRGGBB", result)
}

func TestHexThreeToSixErr(t *testing.T) {
	for _, input := range []string{"", "R", "RG", "RGBA"} {
		t.Run(input, func(t *testing.T) {
			_, err := hexThreeToSix(input)
			assert.Error(t, err)
		})
	}
}

func TestHex(t *testing.T) {
	testCases := []struct {
		input string
		red   int
		green int
		blue  int
	}{
		{"010203", 1, 2, 3},
		{"#010203", 1, 2, 3},
		{"100", 17, 0, 0},
		{"#100", 17, 0, 0},
		{"FFF", 255, 255, 255},
		{"#FFF", 255, 255, 255},
	}

	for _, tc := range testCases {
		red, green, blue, err := hex(tc.input)

		assert.NoError(t, err)
		assert.Equal(t, tc.red, red)
		assert.Equal(t, tc.green, green)
		assert.Equal(t, tc.blue, blue)
	}
}

func TestHexErr(t *testing.T) {
	for _, s := range []string{"", "ZZZ", "0ZZ", "00Z", "1000", "0102GG"} {
		t.Run(s, func(t *testing.T) {
			_, _, _, err := hex(s)
			assert.Error(t, err)
		})
	}
}

func TestRandomColor(t *testing.T) {
	// Get coverage for randomness
	for i := 0; i < 100; i++ {
		color := RandomColor()

		assert.Len(t, color, 7)
		assert.True(t, strings.HasPrefix(color, "#"))
	}
}

func TestIsValidColor(t *testing.T) {
	for _, s := range []string{"FFF", "#FFF", "F0F0F0", "#F0F0F0", "red"} {
		t.Run(s, func(t *testing.T) {
			result := IsValidColor(s)
			assert.True(t, result)
		})
	}
}

func TestIsValidColorErr(t *testing.T) {
	for _, s := range []string{"FF", "#FF", "F0F0", "#F0F0", "ZZZ", "clear"} {
		t.Run(s, func(t *testing.T) {
			result := IsValidColor(s)
			assert.False(t, result)
		})
	}
}
