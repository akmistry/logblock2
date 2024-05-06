package logblock

import (
	"errors"
	"regexp"
	"strconv"
)

var (
	ErrInvalidSizeString = errors.New("invalid size string")

	sizePattern = regexp.MustCompile("^([1-9][0-9]*)([MGTP])?$")
)

func ParseSizeString(str string) (uint64, error) {
	// Special case "0" to simplify the regexp.
	if str == "0" {
		return 0, nil
	}

	parts := sizePattern.FindStringSubmatch(str)
	if len(parts) < 2 {
		return 0, ErrInvalidSizeString
	}

	size, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0, ErrInvalidSizeString
	}
	if len(parts) == 3 {
		switch parts[2] {
		case "M":
			size *= (1 << 20)
		case "G":
			size *= (1 << 30)
		case "T":
			size *= (1 << 40)
		case "P":
			size *= (1 << 50)
		}
	}
	return size, nil
}
