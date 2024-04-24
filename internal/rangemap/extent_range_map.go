package rangemap

import (
	"log"

	"github.com/akmistry/go-util/radix-tree"
)

var _ = (RangeMap[int])((*ExtentRangeMap[int])(nil))

type ExtentRangeMap[V comparable] struct {
	tree radix.Tree
}

func (m *ExtentRangeMap[V]) Begin() (begin uint64, ok bool) {
	m.tree.Ascend(func(i radix.Item) bool {
		ie := i.(*RangeValue[V])
		begin = ie.Offset
		ok = true
		return false
	})
	return
}

func (m *ExtentRangeMap[V]) End() (end uint64) {
	m.tree.Descend(func(i radix.Item) bool {
		ie := i.(*RangeValue[V])
		end = ie.Offset + ie.Length
		return false
	})
	return
}

func (m *ExtentRangeMap[V]) getByteRange(start, length uint64) []*RangeValue[V] {
	end := start + length
	r := Range{Offset: start, Length: length}

	var items []*RangeValue[V]
	m.tree.DescendLessOrEqualI(end, func(i radix.Item) bool {
		ie := i.(*RangeValue[V])
		if ie.Offset == end {
			return true
		} else if !r.Overlaps(ie.Range) {
			return false
		}
		items = append(items, ie)
		return true
	})
	return items
}

func (m *ExtentRangeMap[V]) Get(offset uint64) (value V, ok bool) {
	m.tree.DescendLessOrEqualI(offset, func(i radix.Item) bool {
		ie := i.(*RangeValue[V])
		if ie.Contains(offset) {
			value = ie.Value
			ok = true
		}
		return false
	})
	return
}

func (m *ExtentRangeMap[V]) GetWithRange(offset uint64) (r RangeValue[V], ok bool) {
	m.tree.DescendLessOrEqualI(offset, func(i radix.Item) bool {
		ie := i.(*RangeValue[V])
		if ie.Contains(offset) {
			r = *ie
			ok = true
		}
		return false
	})
	return
}

func (m *ExtentRangeMap[V]) NextKey(offset uint64) (next uint64, ok bool) {
	_, ok = m.Get(offset)
	if ok {
		return offset, ok
	}

	m.tree.AscendGreaterOrEqualI(offset, func(i radix.Item) bool {
		ie := i.(*RangeValue[V])
		next = ie.Offset
		ok = true
		return false
	})
	return
}

func (m *ExtentRangeMap[V]) NextEmpty(offset uint64) (next uint64) {
	next = offset
	m.tree.DescendLessOrEqualI(offset, func(i radix.Item) bool {
		ie := i.(*RangeValue[V])
		if ie.Contains(offset) {
			next = ie.Offset + ie.Length
		}
		return false
	})
	if next == offset {
		return
	}

	m.tree.AscendGreaterOrEqualI(next, func(i radix.Item) bool {
		ie := i.(*RangeValue[V])
		if !ie.Contains(next) {
			return false
		}
		next = ie.Offset + ie.Length
		return true
	})
	return
}

func (m *ExtentRangeMap[V]) Remove(offset, length uint64) {
	if offset < 0 || length < 0 {
		panic("offset < 0 || length < 0")
	} else if length == 0 {
		return
	}

	end := offset + length
	overlaps := m.getByteRange(offset, length)
	for _, ie := range overlaps {
		ieEnd := ie.Offset + ie.Length
		if ie.Offset < offset {
			if ieEnd > end {
				// Old item completely overlaps new item.
				// Split into start and end blocks.
				endItemOffset := end
				endItemLength := ieEnd - end
				endItem := &RangeValue[V]{
					Range: Range{
						Offset: endItemOffset,
						Length: endItemLength,
					},
					Value: ie.Value,
				}
				old := m.tree.ReplaceOrInsert(endItem)
				if old != nil {
					log.Panicf("unexpected old entry: %+v", old)
				}
			}
			// Truncate the old entry instead of deleting it and inserting a new one
			ie.Length = offset - ie.Offset
			ie = nil
		} else if ieEnd > end {
			newItem := &RangeValue[V]{
				Range: Range{
					Offset: end,
					Length: ieEnd - end,
				},
				Value: ie.Value,
			}
			old := m.tree.ReplaceOrInsert(newItem)
			if old != nil {
				log.Panicf("unexpected old entry: %+v", old)
			}
		}

		if ie != nil && m.tree.Delete(ie) != ie {
			log.Panicf("item not deleted: %+v", ie)
		}
	}
}

func (m *ExtentRangeMap[V]) Add(offset, length uint64, value V) {
	if offset < 0 || length < 0 {
		panic("offset < 0 || length < 0")
	} else if length == 0 {
		return
	}

	newItem := &RangeValue[V]{
		Range: Range{Offset: offset, Length: length},
		Value: value,
	}
	// Punch a hole, and put the new item at that hole.
	m.Remove(offset, length)

	old := m.tree.ReplaceOrInsert(newItem)
	if old != nil {
		log.Panicf("unexpected old entry: %+v, adding new entry: %v", old, newItem)
	}
}

func (m *ExtentRangeMap[V]) Iterate(start uint64, iter func(RangeValue[V]) bool) {
	first := start
	if start > 0 {
		m.tree.DescendLessOrEqualI(start, func(i radix.Item) bool {
			ie := i.(*RangeValue[V])
			if ie.Contains(start) {
				first = ie.Offset
			}
			return false
		})
	}

	var r RangeValue[V]
	m.tree.AscendGreaterOrEqualI(first, func(item radix.Item) bool {
		ie := item.(*RangeValue[V])
		if ie.Offset < start {
			r.Offset = start
			r.Length = (ie.Offset + ie.Length) - start
			r.Value = ie.Value
			return true
		}

		if r.Value == ie.Value && ie.Offset == (r.Offset+r.Length) {
			r.Length += ie.Length
		} else {
			if r.Length > 0 && !iter(r) {
				r.Length = 0
				return false
			}
			r = *ie
		}
		return true
	})
	if r.Length > 0 {
		iter(r)
	}
}
