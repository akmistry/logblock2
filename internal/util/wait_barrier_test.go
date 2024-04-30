package util

import (
	"sync"
	"testing"
	"time"
)

func TestWaitBarrier_Basic(t *testing.T) {
	const WaitTime = 10 * time.Millisecond
	var wb WaitBarrier

	// Nothing started, so should not block
	wb.Barrier()

	// Started and finished, so no blocking
	wb.Start().Done()
	wb.Barrier()

	a := wb.Start()
	startTime := time.Now()
	go func() {
		time.Sleep(WaitTime)
		a.Done()
	}()
	wb.Barrier()
	diff := time.Since(startTime)
	if diff < WaitTime {
		t.Errorf("Wait time %v < %v", diff, WaitTime)
	}

	// No new starts, so should not block
	wb.Barrier()
}
func TestWaitBarrier_Ordered(t *testing.T) {
	const WaitTime = 100 * time.Millisecond

	var wb WaitBarrier
	var wg sync.WaitGroup
	wg.Add(2)

	startTime := time.Now()
	a := wb.Start()
	go func() {
		wb.Barrier()
		diff := time.Since(startTime)
		if diff < WaitTime {
			t.Errorf("Wait time %v < %v", diff, WaitTime)
		} else if diff > (11*WaitTime)/10 {
			t.Errorf("Wait time %v > %v", diff, (11*WaitTime)/10)
		}
		wg.Done()
	}()
	time.Sleep(time.Millisecond)

	b := wb.Start()
	go func() {
		time.Sleep(WaitTime)
		a.Done()
		time.Sleep(WaitTime)
		b.Done()
		wg.Done()
	}()

	wb.Barrier()
	end := time.Now()
	if end.Sub(startTime) < 2*WaitTime {
		t.Errorf("Wait time %v < %v", end.Sub(startTime), 2*WaitTime)
	}

	wg.Wait()
}

func TestWaitBarrier_OutOfOrder(t *testing.T) {
	const WaitTime = 100 * time.Millisecond

	var wb WaitBarrier
	var wg sync.WaitGroup
	wg.Add(2)

	startTime := time.Now()
	a := wb.Start()
	go func() {
		wb.Barrier()
		diff := time.Since(startTime)
		if diff < 2*WaitTime {
			t.Errorf("Wait time %v < %v", diff, 2*WaitTime)
		}
		wg.Done()
	}()
	time.Sleep(time.Millisecond)

	b := wb.Start()
	go func() {
		time.Sleep(WaitTime)
		b.Done()
		time.Sleep(WaitTime)
		a.Done()
		wg.Done()
	}()

	wb.Barrier()
	end := time.Now()
	if end.Sub(startTime) < 2*WaitTime {
		t.Errorf("Wait time %v < %v", end.Sub(startTime), 2*WaitTime)
	}

	wg.Wait()
}
