package rangemap

type RangeMap[V any] interface {
	Begin() (begin uint64, ok bool)
	End() (end uint64)

	Add(offset, length uint64, value V)
	// Remove(offset, length uint64)
	Get(offset uint64) (value V, ok bool)

	NextKey(offset uint64) (next uint64, ok bool)
	NextEmpty(offset uint64) (next uint64)

	RangeMapIterator[V]
}

type RangeMapIterator[V any] interface {
	Iterate(start uint64, iter func(RangeValue[V]) bool)
}

type Range struct {
	Offset, Length uint64
}

func (r *Range) Key() uint64 {
	return r.Offset
}

func (r Range) End() uint64 {
	return r.Offset + r.Length
}

func (r Range) Contains(off uint64) bool {
	return off >= r.Offset && off < (r.Offset+r.Length)
}

func (r Range) Overlaps(other Range) bool {
	return r.Contains(other.Offset) || other.Contains(r.Offset)
}

type RangeValue[V any] struct {
	Range
	Value V
}
