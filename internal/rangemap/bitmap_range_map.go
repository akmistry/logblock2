package rangemap

import (
	"log"

	"github.com/akmistry/go-util/bitmap"
	"github.com/bits-and-blooms/bitset"
)

type bitmapRangeMapLeaf[V comparable] struct {
	bm    bitmap.Bitmap256
	items *[256]V
}

func (l *bitmapRangeMapLeaf[V]) empty() bool {
	return l.bm.Empty()
}

func (l *bitmapRangeMapLeaf[V]) full() bool {
	return l.bm.Full()
}

func (l *bitmapRangeMapLeaf[V]) has(i int) bool {
	return l.bm.Get(uint8(i))
}

func (l *bitmapRangeMapLeaf[V]) insert(start, length int, value V) {
	for i := start; i < start+length; i++ {
		l.items[i] = value
		l.bm.Set(uint8(i))
	}
}

var _ = (RangeMap[int])((*BitmapRangeMap[int])(nil))

type BitmapRangeMap[V comparable] struct {
	entries map[uint64]*bitmapRangeMapLeaf[V]

	fullLeafIndex bitset.BitSet
	partLeafIndex bitset.BitSet
}

func (m *BitmapRangeMap[V]) init() {
	if m.entries == nil {
		m.entries = make(map[uint64]*bitmapRangeMapLeaf[V])
	}
}

func NewBitmapRangeMap[V comparable]() *BitmapRangeMap[V] {
	idx := &BitmapRangeMap[V]{
		entries: make(map[uint64]*bitmapRangeMapLeaf[V]),
	}
	return idx
}

func (m *BitmapRangeMap[V]) getLeaf(leafIndex uint64) *bitmapRangeMapLeaf[V] {
	m.init()
	return m.entries[leafIndex]
}

func (m *BitmapRangeMap[V]) getOrCreateLeaf(leafIndex uint64) *bitmapRangeMapLeaf[V] {
	m.init()
	l := m.entries[leafIndex]
	if l == nil {
		l = &bitmapRangeMapLeaf[V]{items: new([256]V)}
		m.entries[leafIndex] = l
	}

	return l
}

func (m *BitmapRangeMap[V]) Begin() (uint64, bool) {
	firstLeafIndex, ok := m.partLeafIndex.NextSet(0)
	if !ok {
		return 0, false
	}
	leaf := m.getLeaf(uint64(firstLeafIndex))
	// |leaf| should always be non-nil here. If it is, let the function panic
	// so we can detect this error.
	ffs := leaf.bm.FindFirstSet()
	if ffs < 256 {
		return uint64(ffs) + (uint64(firstLeafIndex) << 8), true
	}

	log.Panicf("Unexpected empty leaf: %d", firstLeafIndex)
	return 0, false
}

func (m *BitmapRangeMap[V]) End() uint64 {
	endLeafIndex := int(m.partLeafIndex.Len()) - 1
	for ; endLeafIndex >= 0 && !m.partLeafIndex.Test(uint(endLeafIndex)); endLeafIndex-- {
	}
	if endLeafIndex < 0 {
		return 0
	}
	lastLeafIndex := uint64(endLeafIndex)
	leaf := m.getLeaf(lastLeafIndex)
	for i := 255; i >= 0; i-- {
		if leaf.has(i) {
			return uint64(i) + (lastLeafIndex << 8) + 1
		}
	}

	log.Panicf("Unexpected empty leaf: %d", lastLeafIndex)
	return 0
}

func (m *BitmapRangeMap[V]) Add(offset, length uint64, value V) {
	end := offset + length
	for offset < end {
		leafIndex := offset >> 8
		leaf := m.getOrCreateLeaf(leafIndex)

		leafCount := 256 - (offset & 0xFF)
		if leafCount > (end - offset) {
			leafCount = end - offset
		}
		if leaf.empty() {
			m.partLeafIndex.Set(uint(leafIndex))
		}
		leaf.insert(int(offset&0xFF), int(leafCount), value)
		offset += leafCount

		if leaf.full() {
			m.fullLeafIndex.Set(uint(leafIndex))
		}
	}
}

// Not implemented properly, so comment out for now.
/*
func (m *BitmapRangeMap[V]) Remove(offset, length uint64) {
	panic("Unimplemented")
	for length > 0 {
		leaf := m.getLeaf(offset >> 8)
		if leaf != nil {
			wasFull := leaf.full()
			// TODO: Implement me!
			//leaf.items.delete(uint8(offset))
			if leaf.empty() {
				i.entries[offset>>8] = nil
			} else if wasFull {
				// TODO: Implement me!
				//m.fullLeafIndex.Remove(leaf.Key(), 1)
			}
		}

		length--
		offset++
	}
}
*/

func (m *BitmapRangeMap[V]) Get(offset uint64) (value V, ok bool) {
	leaf := m.getLeaf(offset >> 8)
	if leaf == nil {
		return
	}
	if !leaf.has(int(offset & 0xFF)) {
		return
	}
	return leaf.items[uint8(offset)], true
}

func (m *BitmapRangeMap[V]) NextKey(off uint64) (uint64, bool) {
	leaf := m.getLeaf(off >> 8)
	if leaf != nil {
		next := leaf.bm.FindNextSet(uint8(off))
		if next < 256 {
			return (off & ^uint64(0xFF)) + uint64(next), true
		}
	}

	nextPartial, ok := m.partLeafIndex.NextSet(uint(off>>8) + 1)
	if !ok {
		return 0, false
	}

	leaf = m.entries[uint64(nextPartial)]
	return uint64(nextPartial<<8) + uint64(leaf.bm.FindFirstSet()), true
}

func (m *BitmapRangeMap[V]) NextEmpty(off uint64) uint64 {
	leaf := m.getLeaf(off >> 8)
	if leaf == nil {
		return off
	}
	next := leaf.bm.FindNextClear(uint8(off))
	if next < 256 {
		return (off & ^uint64(0xFF)) + uint64(next)
	}

	clearStart := uint(off>>8) + 1
	nextNonFull, ok := m.fullLeafIndex.NextClear(uint(clearStart))
	if !ok {
		if clearStart < m.fullLeafIndex.Len() {
			nextNonFull = m.fullLeafIndex.Len()
		} else {
			nextNonFull = clearStart
		}
	}

	nextOff := uint64(nextNonFull << 8)
	leaf = m.getLeaf(uint64(nextNonFull))
	if leaf == nil {
		return nextOff
	}

	if leaf.full() {
		panic("Unexpected full leaf")
	}

	return nextOff + uint64(leaf.bm.FindFirstClear())
}

func (m *BitmapRangeMap[V]) Iterate(start uint64, iter func(RangeValue[V]) bool) {
	off := start
	var r RangeValue[V]
	for {
		leaf := m.getLeaf(off >> 8)
		if leaf == nil {
			next, ok := m.NextKey(off)
			if !ok {
				break
			}
			off = next
			continue
		}

		for j := int(off & 0xFF); j < 256; j++ {
			if !leaf.has(j) {
				off++
				continue
			}
			v := leaf.items[j]
			if r.Value == v && (r.Offset+r.Length) == off {
				// Coalesce consecutive blocks
				r.Length++
			} else {
				// New range. First give the current one.
				if r.Length > 0 && !iter(r) {
					r.Length = 0
					return
				}
				r.Offset = off
				r.Length = 1
				r.Value = v
			}
			off++
		}
	}
	if r.Length > 0 {
		iter(r)
	}
}
