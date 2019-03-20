package common

import (
	"regexp"
	"strings"
)

var colors = []string{
	"AliceBlue", "AntiqueWhite", "Aqua", "Aquamarine", "Azure",
	"Beige", "Bisque", "Black", "BlanchedAlmond", "Blue",
	"BlueViolet", "Brown", "BurlyWood", "CadetBlue", "Chartreuse",
	"Chocolate", "Coral", "CornflowerBlue", "Cornsilk", "Crimson",
	"Cyan", "DarkBlue", "DarkCyan", "DarkGoldenRod", "DarkGray",
	"DarkGrey", "DarkGreen", "DarkKhaki", "DarkMagenta", "DarkOliveGreen",
	"DarkOrange", "DarkOrchid", "DarkRed", "DarkSalmon", "DarkSeaGreen",
	"DarkSlateBlue", "DarkSlateGray", "DarkSlateGrey", "DarkTurquoise", "DarkViolet",
	"DeepPink", "DeepSkyBlue", "DimGray", "DimGrey", "DodgerBlue",
	"FireBrick", "FloralWhite", "ForestGreen", "Fuchsia", "Gainsboro",
	"GhostWhite", "Gold", "GoldenRod", "Gray", "Grey",
	"Green", "GreenYellow", "HoneyDew", "HotPink", "IndianRed",
	"Indigo", "Ivory", "Khaki", "Lavender", "LavenderBlush",
	"LawnGreen", "LemonChiffon", "LightBlue", "LightCoral", "LightCyan",
	"LightGoldenRodYellow", "LightGray", "LightGrey", "LightGreen", "LightPink",
	"LightSalmon", "LightSeaGreen", "LightSkyBlue", "LightSlateGray", "LightSlateGrey",
	"LightSteelBlue", "LightYellow", "Lime", "LimeGreen", "Linen",
	"Magenta", "Maroon", "MediumAquaMarine", "MediumBlue", "MediumOrchid",
	"MediumPurple", "MediumSeaGreen", "MediumSlateBlue", "MediumSpringGreen", "MediumTurquoise",
	"MediumVioletRed", "MidnightBlue", "MintCream", "MistyRose", "Moccasin",
	"NavajoWhite", "Navy", "OldLace", "Olive", "OliveDrab",
	"Orange", "OrangeRed", "Orchid", "PaleGoldenRod", "PaleGreen",
	"PaleTurquoise", "PaleVioletRed", "PapayaWhip", "PeachPuff", "Peru",
	"Pink", "Plum", "PowderBlue", "Purple", "RebeccaPurple",
	"Red", "RosyBrown", "RoyalBlue", "SaddleBrown", "Salmon",
	"SandyBrown", "SeaGreen", "SeaShell", "Sienna", "Silver",
	"SkyBlue", "SlateBlue", "SlateGray", "SlateGrey", "Snow",
	"SpringGreen", "SteelBlue", "Tan", "Teal", "Thistle",
	"Tomato", "Turquoise", "Violet", "Wheat", "White",
	"WhiteSmoke", "Yellow", "YellowGreen",
}

// IsValidColor takes a string s and compares it against a list of css color names.
// It also accepts hex codes in the form of #000 (RGB), to #00000000 (RRGGBBAA), with A
// being the alpha value
func IsValidColor(s string) bool {
	for _, c := range colors {
		if strings.ToLower(c) == strings.ToLower(s) {
			return true
		}
	}

	return regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`).MatchString(s)
}
