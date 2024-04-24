package rangemap

import (
	"math/rand"
	"testing"
)

type testRangeMap = RangeMap[int]

func checkBeginEnd(t *testing.T, m testRangeMap, expBegin uint64, expOk bool, expEnd uint64) {
	t.Helper()
	begin, ok := m.Begin()
	if ok != expOk || begin != expBegin {
		t.Errorf("Unexpected Begin() (%d, %v) != (%d, %v)", begin, ok, expBegin, expOk)
	}
	end := m.End()
	if end != expEnd {
		t.Errorf("Unexpected End() %d != %d", end, expEnd)
	}
}

func testBeginEnd(t *testing.T, m testRangeMap) {
	checkBeginEnd(t, m, 0, false, 0)

	m.Add(1234, 123, 1)
	checkBeginEnd(t, m, 1234, true, 1357)
	m.Add(1230, 4, 2)
	checkBeginEnd(t, m, 1230, true, 1357)
	m.Add(1230, 1, 3)
	checkBeginEnd(t, m, 1230, true, 1357)
	m.Add(1229, 1, 4)
	checkBeginEnd(t, m, 1229, true, 1357)

	m.Add(1345, 5, 5)
	checkBeginEnd(t, m, 1229, true, 1357)
	m.Add(1350, 9, 6)
	checkBeginEnd(t, m, 1229, true, 1359)

	// Stress test
	const MaxOffset = 100000
	const MaxLength = 1000
	begin, _ := m.Begin()
	end := m.End()
	for i := 0; i < 1000; i++ {
		off := uint64(rand.Int63n(MaxOffset))
		length := uint64(rand.Int63n(MaxLength) + 1)
		m.Add(off, length, i)
		if off < begin {
			begin = off
		}
		if (off + length) > end {
			end = off + length
		}
		checkBeginEnd(t, m, begin, true, end)
	}
}

func TestExtentRangeMap_BeginEnd(t *testing.T) {
	var m ExtentRangeMap[int]
	testBeginEnd(t, &m)
}

func TestBitmapRangeMap_BeginEnd(t *testing.T) {
	var m BitmapRangeMap[int]
	testBeginEnd(t, &m)
}

func testAddGet(t *testing.T, m testRangeMap) {
	const RangeLength = 100000
	values := make([]int, RangeLength)

	const MaxLength = 1000
	const Iterations = 100
	for i := 1; i < Iterations; i++ {
		off := uint64(rand.Int63n(RangeLength - MaxLength))
		length := uint64(rand.Int63n(MaxLength) + 1)
		m.Add(off, length, i)
		for j := uint64(0); j < length; j++ {
			values[off+j] = i
		}

		for j, v := range values {
			gv, ok := m.Get(uint64(j))
			if v == 0 {
				if ok {
					t.Errorf("Get(%d) expected !ok", j)
				}
			} else {
				if !ok || gv != v {
					t.Errorf("Get(%d) (%d, %v) != (%d, true)", j, gv, ok, v)
				}
			}
		}
	}
}

func TestExtentRangeMap_AddGet(t *testing.T) {
	var m ExtentRangeMap[int]
	testAddGet(t, &m)
}

func TestBitmapRangeMap_AddGet(t *testing.T) {
	var m BitmapRangeMap[int]
	testAddGet(t, &m)
}

func testNext(t *testing.T, m testRangeMap) {
	const RangeLength = 100000
	values := make([]int, RangeLength)

	const MaxLength = 1000
	const Iterations = 100
	for i := 1; i < Iterations; i++ {
		off := uint64(rand.Int63n(RangeLength - MaxLength))
		length := uint64(rand.Int63n(MaxLength) + 1)
		m.Add(off, length, i)
		for j := uint64(0); j < length; j++ {
			values[off+j] = i
		}
	}

	for i, v := range values {
		nextKey, ok := m.NextKey(uint64(i))
		nextEmpty := m.NextEmpty(uint64(i))
		if v == 0 {
			if nextEmpty != uint64(i) {
				t.Errorf("NextEmpty(%d) %d != %d", i, nextEmpty, i)
			}

			// Find next value
			j := uint64(i)
			for ; j < RangeLength && values[j] == 0; j++ {
			}
			if j >= RangeLength {
				// No next
				if ok {
					t.Errorf("NextKey(%d) ok", i)
				}
			} else {
				if !ok || nextKey != j {
					t.Errorf("NextKey(%d) (%d, %v) != (%d, true)", i, nextKey, ok, j)
				}
			}
		} else {
			if !ok || nextKey != uint64(i) {
				t.Errorf("NextKey(%d) (%d, %v) != (%d, true)", i, nextKey, ok, i)
			}

			// Find next empty
			j := uint64(i)
			for ; j < RangeLength && values[j] != 0; j++ {
			}
			if nextEmpty != j {
				t.Errorf("NextEmpty(%d) %d != %d", i, nextEmpty, j)
			}
		}
	}
}

func TestExtentRangeMap_Next(t *testing.T) {
	var m ExtentRangeMap[int]
	testNext(t, &m)
}

func TestBitmapRangeMap_Next(t *testing.T) {
	var m BitmapRangeMap[int]
	testNext(t, &m)
}

func testIterate(t *testing.T, m testRangeMap) {
	const RangeLength = 10000
	values := make([]int, RangeLength)

	const MaxLength = 1000
	const Iterations = 10
	for i := 1; i < Iterations; i++ {
		off := uint64(rand.Int63n(RangeLength - MaxLength))
		length := uint64(rand.Int63n(MaxLength) + 1)
		m.Add(off, length, i)
		for j := uint64(0); j < length; j++ {
			values[off+j] = i
		}
	}

	for start := range values {
		prevEnd := uint64(0)
		valueCount := uint64(0)
		m.Iterate(uint64(start), func(r RangeValue[int]) bool {
			if r.Offset < uint64(start) {
				t.Errorf("Offset %d < start %d", r.Offset, start)
			}
			if r.Offset < prevEnd {
				t.Errorf("Offset %d < prevEnd %d", r.Offset, prevEnd)
			}

			for i := r.Offset; i < r.End(); i++ {
				if values[i] != r.Value {
					t.Errorf("values[%d] %d != r.Value %d", i, values[i], r.Value)
				}
			}

			prevEnd = r.Offset + r.Length
			valueCount += r.Length
			return true
		})
		end := m.End()
		if uint64(start) < end && prevEnd != end {
			t.Errorf("Iterate end %d != End() %d", prevEnd, end)
		}

		actualValues := uint64(0)
		for i := start; i < RangeLength; i++ {
			if values[i] != 0 {
				actualValues++
			}
		}
		if valueCount != actualValues {
			t.Errorf("valueCount %d != actual %d", valueCount, actualValues)
		}
	}
}

func TestExtentRangeMap_Iterate(t *testing.T) {
	var m ExtentRangeMap[int]
	testIterate(t, &m)
}

func TestBitmapRangeMap_Iterate(t *testing.T) {
	var m BitmapRangeMap[int]
	testIterate(t, &m)
}

func benchmarkGet(b *testing.B, m testRangeMap) {
	const RangeLength = 1000000
	values := make([]int, RangeLength)

	const MaxLength = 1000
	const Iterations = 1000
	for i := 1; i < Iterations; i++ {
		off := uint64(rand.Int63n(RangeLength - MaxLength))
		length := uint64(rand.Int63n(MaxLength) + 1)
		m.Add(off, length, i)
		for j := uint64(0); j < length; j++ {
			values[off+j] = i
		}
	}

	randOffsets := make([]uint64, b.N)
	for i := range randOffsets {
		randOffsets[i] = uint64(rand.Int63n(RangeLength))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Get(randOffsets[i])
	}
}

func BenchmarkExtentRangeMap_Get(b *testing.B) {
	var m ExtentRangeMap[int]
	benchmarkGet(b, &m)
}

func BenchmarkBitmapRangeMap_Get(b *testing.B) {
	var m BitmapRangeMap[int]
	benchmarkGet(b, &m)
}

func benchmarkAdd(b *testing.B, m testRangeMap) {
	const RangeLength = 1000000
	const MaxLength = 512

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		off := uint64(rand.Int63n(RangeLength - MaxLength))
		length := uint64(rand.Int63n(MaxLength) + 1)
		m.Add(off, length, i+1)
	}
}

func BenchmarkExtentRangeMap_Add(b *testing.B) {
	var m ExtentRangeMap[int]
	benchmarkAdd(b, &m)
}

func BenchmarkBitmapRangeMap_Add(b *testing.B) {
	var m BitmapRangeMap[int]
	benchmarkAdd(b, &m)
}
