package common

import (
	"testing"
)

func TestColorHexThreeToSix(t *testing.T) {
	expected := "RRGGBB"
	result, _ := hexThreeToSix("RGB")
	if result != expected {
		t.Errorf("expected %#v, got %#v", expected, result)
	}
}

func TestHex(t *testing.T) {
	// The testing data layout is inputer, Expected Red, Exp Green, Exp Blue, expect error
	data := [][]interface{}{
		[]interface{}{"010203", 1, 2, 3, false},
		[]interface{}{"100", 17, 0, 0, false},
		[]interface{}{"100", 1, 0, 0, true},
		[]interface{}{"1000", 0, 0, 0, true},
		[]interface{}{"010203", 1, 2, 4, true},
		[]interface{}{"0102GG", 1, 2, 4, true},
	}

	for i := range data {
		input := data[i][0].(string)
		r, g, b, err := hex(input)
		if err != nil {
			if !data[i][4].(bool) {
				t.Errorf("with input %#v: %v", input, err)
			}
			continue
		}

		rr, rg, rb := data[i][1].(int), data[i][2].(int), data[i][3].(int)

		if !data[i][4].(bool) && (r != rr || g != rg || b != rb) {
			t.Errorf("expected %d, %d, %d - got %d, %d, %d", r, g, b, rr, rg, rb)
		}
	}
}
