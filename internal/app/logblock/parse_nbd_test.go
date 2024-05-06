package logblock

import (
	"testing"
)

func TestParseNbdIndex(t *testing.T) {
	tests := []struct {
		str    string
		expInt int
		expErr bool
	}{
		{"", 0, true},
		{"/dev/nbd0", 0, false},
		{"/dev/nbd7", 7, false},
		{"/dev/nbd123", 123, false},
		{"/dev/nbd", 0, true},
		{"/dev/nbda", 0, true},
		{"/dev/nbe", 0, true},
		{"some/random/path", 0, true},
	}
	for _, tc := range tests {
		i, err := ParseNbdIndex(tc.str)
		if i != tc.expInt {
			t.Errorf("ParseNbdIndex(%s) %d != %d", tc.str, i, tc.expInt)
		}
		if tc.expErr {
			if err == nil {
				t.Errorf("ParseNbdIndex(%s) unexpected nil error", tc.str)
			}
		} else {
			if err != nil {
				t.Errorf("ParseNbdIndex(%s) unexpected error %v", tc.str, err)
			}
		}
	}
}
