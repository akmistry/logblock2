package logblock

import (
	"testing"
)

const (
	megabyte = 1024 * 1024
	gigabyte = 1024 * megabyte
	terabyte = 1024 * gigabyte
	petabyte = 1024 * terabyte
)

func TestParseSizeString(t *testing.T) {
	tests := []struct {
		str    string
		expInt uint64
		expErr bool
	}{
		{"0", 0, false},
		{"1", 1, false},
		{"123456", 123456, false},
		{"12M", 12 * megabyte, false},
		{"2G", 2 * gigabyte, false},
		{"345T", 345 * terabyte, false},
		{"3P", 3 * petabyte, false},
		{"12a3", 0, true},
		{"01", 0, true},
		{"0T", 0, true},
		{"1 T", 0, true},
		{"1Gb", 0, true},
		{"1E", 0, true},
		// TODO: Consider support leading and trailing whitespace.
		{" 1G", 0, true},
	}
	for _, tc := range tests {
		i, err := ParseSizeString(tc.str)
		if i != tc.expInt {
			t.Errorf("ParseSizeString(%s) %d != %d", tc.str, i, tc.expInt)
		}
		if tc.expErr {
			if err == nil {
				t.Errorf("ParseSizeString(%s) unexpected nil error", tc.str)
			}
		} else {
			if err != nil {
				t.Errorf("ParseSizeString(%s) unexpected error %v", tc.str, err)
			}
		}
	}
}
