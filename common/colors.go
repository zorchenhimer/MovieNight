package common

import (
	"errors"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func init() {
	rand.Seed(int64(time.Now().Nanosecond()))
}

// Colors holds all the valid html color names for MovieNight
// the values in colors must be lowercase so it matches with the color input
// this saves from having to call strings.ToLower(color) every time to check
var Colors = []string{
	"aliceblue",
	"antiquewhite",
	"aqua",
	"aquamarine",
	"azure",
	"beige",
	"bisque",
	"blanchedalmond",
	"burlywood",
	"cadetblue",
	"chartreuse",
	"chocolate",
	"coral",
	"cornflowerblue",
	"cornsilk",
	"cyan",
	"darkcyan",
	"darkgoldenrod",
	"darkgray",
	"darkkhaki",
	"darkorange",
	"darksalmon",
	"darkseagreen",
	"darkturquoise",
	"deeppink",
	"deepskyblue",
	"dodgerblue",
	"floralwhite",
	"fuchsia",
	"gainsboro",
	"ghostwhite",
	"gold",
	"goldenrod",
	"gray",
	"greenyellow",
	"honeydew",
	"hotpink",
	"ivory",
	"khaki",
	"lavender",
	"lavenderblush",
	"lawngreen",
	"lemonchiffon",
	"lightblue",
	"lightcoral",
	"lightcyan",
	"lightgoldenrodyellow",
	"lightgreen",
	"lightgrey",
	"lightpink",
	"lightsalmon",
	"lightseagreen",
	"lightskyblue",
	"lightslategray",
	"lightsteelblue",
	"lightyellow",
	"lime",
	"limegreen",
	"linen",
	"magenta",
	"mediumaquamarine",
	"mediumorchid",
	"mediumpurple",
	"mediumseagreen",
	"mediumslateblue",
	"mediumspringgreen",
	"mediumturquoise",
	"mintcream",
	"mistyrose",
	"moccasin",
	"navajowhite",
	"oldlace",
	"olive",
	"olivedrab",
	"orange",
	"orangered",
	"orchid",
	"palegoldenrod",
	"palegreen",
	"paleturquoise",
	"palevioletred",
	"papayawhip",
	"peachpuff",
	"peru",
	"pink",
	"plum",
	"powderblue",
	"red",
	"rosybrown",
	"salmon",
	"sandybrown",
	"seagreen",
	"seashell",
	"silver",
	"skyblue",
	"slategray",
	"snow",
	"springgreen",
	"steelblue",
	"tan",
	"thistle",
	"tomato",
	"turquoise",
	"violet",
	"wheat",
	"white",
	"whitesmoke",
	"yellow",
	"yellowgreen",
}

var (
	regexColor = regexp.MustCompile(`^([0-9A-Fa-f]{3}){1,2}$`)
)

// IsValidColor takes a string s and compares it against a list of css color names.
// It also accepts hex codes in the form of #RGB and #RRGGBB
func IsValidColor(s string) bool {
	s = strings.TrimLeft(strings.ToLower(s), "#")
	for _, c := range Colors {
		if s == c {
			return true
		}
	}

	if regexColor.MatchString(s) {
		r, g, b, err := hex(s)
		if err != nil {
			return false
		}
		total := float32(r + g + b)
		return total > 0.7 && float32(b)/total < 0.7
	}
	return false
}

// RandomColor returns a hex color code
func RandomColor() string {
	var color string
	for !IsValidColor(color) {
		color = ""
		for i := 0; i < 3; i++ {
			s := strconv.FormatInt(rand.Int63n(255), 16)
			if len(s) == 1 {
				s = "0" + s
			}
			color += s
		}
	}
	return "#" + color
}

// hex returns R, G, B as values
func hex(s string) (int, int, int, error) {
	// Make the string just the base16 numbers
	s = strings.TrimLeft(s, "#")

	if len(s) == 3 {
		var err error
		s, err = hexThreeToSix(s)
		if err != nil {
			return 0, 0, 0, err
		}
	}

	if len(s) == 6 {
		R64, err := strconv.ParseInt(s[0:2], 16, 32)
		if err != nil {
			return 0, 0, 0, err
		}

		G64, err := strconv.ParseInt(s[2:4], 16, 32)
		if err != nil {
			return 0, 0, 0, err
		}

		B64, err := strconv.ParseInt(s[4:6], 16, 32)
		if err != nil {
			return 0, 0, 0, err
		}

		return int(R64), int(G64), int(B64), nil
	}
	return 0, 0, 0, errors.New("incorrect format")
}

func hexThreeToSix(s string) (string, error) {
	if len(s) != 3 {
		return "", fmt.Errorf("%d is the incorrect length of string for convertsion", len(s))
	}

	h := ""
	for i := 0; i < 3; i++ {
		h += string(s[i])
		h += string(s[i])
	}
	return h, nil
}
