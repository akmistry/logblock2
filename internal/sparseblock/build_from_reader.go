package sparseblock

import (
	"fmt"
	"io"
	"log/slog"
	"math"
	"time"

	"github.com/akmistry/logblock2/internal/t"
	"github.com/akmistry/logblock2/internal/util"
)

const (
	lowMaxDataSizeThreshold = 1024 * 1024
)

type BuildOptions struct {
	MaxDataSize int64
}

func BuildFromReader(dst io.Writer, src t.HoleReaderAt, blockSize int, opts *BuildOptions) error {
	maxDataSize := int64(math.MaxInt64)
	if opts != nil && opts.MaxDataSize > 0 {
		maxDataSize = opts.MaxDataSize
	}
	if maxDataSize < lowMaxDataSizeThreshold {
		// Warn the user if MaxDataSize is set too low.
		slog.Warn(fmt.Sprintf("sparseblock/BuildFromReader: MaxDataSize less than suggested minimum %d",
			lowMaxDataSizeThreshold))
	}

	b := NewBuilder(dst, blockSize)
	offset := int64(0)
	dataBytes := int64(0)
	addStart := time.Now()
	count := 0
	for dataBytes < maxDataSize {
		startOffset, err := src.NextData(offset)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		endOffset, err := src.NextHole(startOffset)
		if err != nil {
			panic(err)
		}

		length := endOffset - startOffset
		if dataBytes+length > maxDataSize {
			length = maxDataSize - dataBytes
			endOffset = startOffset + length
			if length < int64(blockSize) {
				break
			}
		}

		if startOffset%int64(blockSize) != 0 || length%int64(blockSize) != 0 {
			return fmt.Errorf("(offset %d, length %d) not a multiple of blockSize %d",
				startOffset, length, blockSize)
		}

		b.AddBlocks(startOffset/int64(blockSize), length/int64(blockSize), io.NewSectionReader(src, startOffset, length))
		offset = endOffset
		dataBytes += length
		count++
	}

	buildStart := time.Now()
	defer func() {
		addTime := buildStart.Sub(addStart)
		buildTime := time.Since(buildStart)
		slog.Info("sparseblock/BuildFromReader: Build done",
			"entries", count,
			"addTime", addTime,
			"addTimePerEntry", addTime/time.Duration(count),
			"buildTime", buildTime,
			"buildTimePerEntry", buildTime/time.Duration(count),
			"bytesPerSec", int64(float64(dataBytes)/buildTime.Seconds()),
			"bw", util.DetailedBytes(float64(dataBytes)/buildTime.Seconds()))
	}()

	err := b.Build()

	return err
}
