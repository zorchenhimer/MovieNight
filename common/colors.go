package common

import (
	"regexp"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
)

// Colors holds all the valid html color names for MovieNight
// the values in colors must be lowercase so it matches with the color input
// this saves from having to call strings.ToLower(color) every time to check
var Colors = []string{
	"aliceblue", "antiquewhite", "aqua", "aquamarine", "azure",
	"beige", "bisque", "blanchedalmond", "blueviolet", "brown",
	"burlywood", "cadetblue", "chartreuse", "chocolate", "coral",
	"cornflowerblue", "cornsilk", "crimson", "cyan", "darkcyan",
	"darkgoldenrod", "darkgray", "darkkhaki", "darkmagenta", "darkolivegreen",
	"darkorange", "darkorchid", "darksalmon", "darkseagreen", "darkslateblue",
	"darkslategray", "darkturquoise", "darkviolet", "deeppink", "deepskyblue",
	"dimgray", "dodgerblue", "firebrick", "floralwhite", "forestgreen",
	"fuchsia", "gainsboro", "ghostwhite", "gold", "goldenrod",
	"gray", "greenyellow", "honeydew", "hotpink", "indigo",
	"ivory", "khaki", "lavender", "lavenderblush", "lawngreen",
	"lemonchiffon", "lightblue", "lightcoral", "lightcyan", "lightgoldenrodyellow",
	"lightgrey", "lightgreen", "lightpink", "lightsalmon", "lightseagreen",
	"lightskyblue", "lightslategray", "lightsteelblue", "lightyellow", "lime",
	"limegreen", "linen", "magenta", "mediumaquamarine", "mediumorchid",
	"mediumpurple", "mediumseagreen", "mediumslateblue", "mediumspringgreen", "mediumturquoise",
	"mediumvioletred", "mintcream", "mistyrose", "moccasin", "navajowhite",
	"oldlace", "olive", "olivedrab", "orange", "orangered",
	"orchid", "palegoldenrod", "palegreen", "paleturquoise", "palevioletred",
	"papayawhip", "peachpuff", "peru", "pink", "plum",
	"powderblue", "purple", "rebeccapurple", "red", "rosybrown",
	"royalblue", "saddlebrown", "salmon", "sandybrown", "seagreen",
	"seashell", "sienna", "silver", "skyblue", "slateblue",
	"slategray", "snow", "springgreen", "steelblue", "tan",
	"teal", "thistle", "tomato", "turquoise", "violet",
	"wheat", "white", "whitesmoke", "yellow", "yellowgreen",
}

var (
	regexColor = regexp.MustCompile(`^#([0-9A-Fa-f]{3}){1,2}$`)
)

// IsValidColor takes a string s and compares it against a list of css color names.
// It also accepts hex codes in the form of #RGB and #RRGGBB
func IsValidColor(s string) bool {
	s = strings.ToLower(s)
	for _, c := range Colors {
		if s == c {
			return true
		}
	}

	if regexColor.MatchString(s) {
		c, err := colorful.Hex(s)
		if err != nil {
			return false
		}
		total := c.R + c.G + c.B
		return total > 0.7 && c.B/total < 0.7
	}
	return false
}

// RandomColor returns a hex color code
func RandomColor() string {
	var color colorful.Color
	for !IsValidColor(color.Hex()) {
		color = colorful.FastHappyColor()
	}
	return color.Hex()
}
