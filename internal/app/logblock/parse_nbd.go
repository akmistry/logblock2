package logblock

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidNbdPath = errors.New("invalid nbd path")
)

const (
	nbdPrefix = "/dev/nbd"
)

func ParseNbdIndex(dev string) (int, error) {
	if !strings.HasPrefix(dev, nbdPrefix) {
		return 0, ErrInvalidNbdPath
	}
	i, err := strconv.ParseUint(dev[len(nbdPrefix):], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("error parsing nbd device: %w", err)
	}
	return int(i), nil
}
