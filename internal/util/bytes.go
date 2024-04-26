package util

import (
	"fmt"
)

var suffixes = []string{"Bytes", "KB", "MB", "GB", "TB", "PB", "EB", "ZB"}

func humanReadableBytes(b uint64) string {
	v := float64(b)
	pow := 0
	for v >= 1024 {
		pow++
		v /= 1024
	}

	if v < 10 {
		return fmt.Sprintf("%0.2f %s", v, suffixes[pow])
	} else if v < 100 {
		return fmt.Sprintf("%0.1f %s", v, suffixes[pow])
	}
	return fmt.Sprintf("%0.0f %s", v, suffixes[pow])
}

type Bytes uint64

func (b Bytes) String() string {
	return humanReadableBytes(uint64(b))
}

type DetailedBytes uint64

func (b DetailedBytes) String() string {
	return fmt.Sprintf("%s (%d bytes)", humanReadableBytes(uint64(b)), b)
}
