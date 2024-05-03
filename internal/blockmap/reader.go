package blockmap

import (
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/bits-and-blooms/bitset"

	"github.com/akmistry/logblock2/internal/rangemap"
	"github.com/akmistry/logblock2/internal/t"
)

type Reader struct {
	blockSize int
	numBlocks int64

	rangeMap *rangemap.BitmapRangeMap[uint16]

	valueMap map[uint16]t.BlockHoleReader
	indexMap map[t.BlockHoleReader]uint16
	freeSet  *bitset.BitSet

	rs []t.BlockHoleReader

	lock sync.RWMutex
}

var _ = (t.BlockHoleReader)((*Reader)(nil))

func NewReader(blockSize int, numBlocks int64) *Reader {
	r := &Reader{
		blockSize: blockSize,
		numBlocks: numBlocks,
		rangeMap:  rangemap.NewBitmapRangeMap[uint16](),
		valueMap:  make(map[uint16]t.BlockHoleReader),
		indexMap:  make(map[t.BlockHoleReader]uint16),
		freeSet:   bitset.New(1 << 16),
	}
	r.freeSet.SetAll().Clear(0)
	return r
}

func (r *Reader) NextBlockData(off int64) (nextBlock int64, err error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	next, ok := r.rangeMap.NextKey(uint64(off))
	if !ok {
		return 0, io.EOF
	}
	return int64(next), nil
}

func (r *Reader) NextBlockHole(off int64) (nextHole int64, err error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return int64(r.rangeMap.NextEmpty(uint64(off))), nil
}

func (r *Reader) fetchOrAddValueIndex(br t.BlockHoleReader) uint16 {
	iv, ok := r.indexMap[br]
	if ok {
		return iv
	}

	nv, ok := r.freeSet.NextSet(1)
	if !ok {
		panic("Index full")
	}
	r.freeSet.Clear(nv)
	r.valueMap[uint16(nv)] = br
	r.indexMap[br] = uint16(nv)
	return uint16(nv)
}

func (r *Reader) PushReader(br t.BlockHoleReader) {
	const BatchSize = 4096

	r.lock.Lock()
	for _, hr := range r.rs {
		if br == hr {
			panic("Appending reader already in index")
		}
	}
	r.rs = append(r.rs, br)
	iv := r.fetchOrAddValueIndex(br)
	r.lock.Unlock()

	type blockRange struct {
		offset, length int64
	}
	rl := make([]blockRange, 0, BatchSize)
	doAdd := func() {
		r.lock.Lock()
		defer r.lock.Unlock()
		for _, rg := range rl {
			r.rangeMap.Add(uint64(rg.offset), uint64(rg.length), iv)
		}
	}
	defer doAdd()
	addRange := func(offset, length int64) {
		rl = append(rl, blockRange{offset: offset, length: length})
		if len(rl) >= BatchSize {
			doAdd()
			rl = rl[:0]
		}
	}

	var offset int64
	for {
		startOffset, err := br.NextBlockData(offset)
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		endOffset, err := br.NextBlockHole(startOffset)
		if err != nil {
			panic(err)
		}
		length := endOffset - startOffset

		addRange(startOffset, length)
		offset = endOffset
	}
}

func (r *Reader) RemoveTable(br t.BlockHoleReader) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for j, rs := range r.rs {
		if rs == br {
			r.rs = slices.Delete(r.rs, j, j+1)
			break
		}
	}

	iv := r.indexMap[br]
	if iv == 0 {
		panic("iv == 0")
	}
	delete(r.indexMap, br)
	delete(r.valueMap, iv)
	r.freeSet.Set(uint(iv))
}

func (r *Reader) ListReaders() []t.BlockHoleReader {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return slices.Clone(r.rs)
}

func (r *Reader) LiveBlocks() int64 {
	r.lock.RLock()
	defer r.lock.RUnlock()

	startTime := time.Now()
	count := uint64(0)
	r.rangeMap.Iterate(0, func(rg rangemap.RangeValue[uint16]) bool {
		count += rg.Length
		return true
	})

	slog.Info("Live blocks (count only)", "count", count, "time", time.Since(startTime))

	return int64(count)
}

func (r *Reader) GetLiveBlocks() map[t.BlockHoleReader]int64 {
	r.lock.RLock()
	defer r.lock.RUnlock()

	startTime := time.Now()

	blockCount := make(map[uint16]int64)
	r.rangeMap.Iterate(0, func(rg rangemap.RangeValue[uint16]) bool {
		blockCount[rg.Value] += int64(rg.Length)
		return true
	})

	liveBlocks := make(map[t.BlockHoleReader]int64)
	for _, br := range r.rs {
		liveBlocks[br] = 0
	}
	count := int64(0)
	for iv, c := range blockCount {
		liveBlocks[r.valueMap[iv]] = c
		count += c
	}

	slog.Info("Live blocks (readers)", "count", count, "time", time.Since(startTime))
	return liveBlocks
}

func (r *Reader) ReaderForLiveBytes(rs []t.BlockHoleReader) *Reader {
	r.lock.RLock()
	defer r.lock.RUnlock()

	newReader := NewReader(r.blockSize, r.numBlocks)
	newReader.rs = slices.Clone(rs)

	readers := make(map[uint16]bool)
	for _, br := range rs {
		iv, ok := r.indexMap[br]
		if !ok {
			panic("reader not present")
		}
		readers[iv] = true

		newReader.freeSet.Clear(uint(iv))
		newReader.valueMap[iv] = br
		newReader.indexMap[br] = iv
	}

	r.rangeMap.Iterate(0, func(rg rangemap.RangeValue[uint16]) bool {
		i := rg.Value
		if readers[i] {
			newReader.rangeMap.Add(rg.Offset, rg.Length, i)
		}
		return true
	})

	return newReader
}

func (r *Reader) find(off int64) t.BlockHoleReader {
	iv, ok := r.rangeMap.Get(uint64(off))
	if !ok {
		return nil
	}
	return r.valueMap[iv]
}

func (r *Reader) ReadBlocks(b []byte, off int64) (int, error) {
	if len(b)%r.blockSize != 0 {
		return 0, fmt.Errorf("blockmap.Reader: invalid read size %d, block size %d", len(b), r.blockSize)
	}

	r.lock.RLock()
	defer r.lock.RUnlock()

	n := 0
	for len(b) > 0 {
		br := r.find(off)
		if br == nil {
			clear(b[:r.blockSize])
			n++
			off++
			b = b[r.blockSize:]
		} else {
			rem := (len(b) / r.blockSize) - 1
			readSizeBlocks := 1
			for ; rem > 0; rem-- {
				nextR := r.find(off + int64(readSizeBlocks))
				if nextR == nil || nextR != br {
					break
				}
				readSizeBlocks++
			}

			r.lock.RUnlock()
			blocksRead, err := br.ReadBlocks(b[:readSizeBlocks*r.blockSize], off)
			r.lock.RLock()
			n += blocksRead
			off += int64(blocksRead)
			if err != nil {
				return n, err
			}
			b = b[blocksRead*r.blockSize:]
		}
	}

	return n, nil
}
