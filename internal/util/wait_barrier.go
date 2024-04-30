package util

import (
	"sync"
)

type Unlocker interface {
	Unlock()
}

type WaitBarrierContext struct {
	wg *sync.WaitGroup
}

func (c WaitBarrierContext) Done() {
	c.wg.Done()
}

type WaitBarrier struct {
	wg   *sync.WaitGroup
	lock sync.Mutex
}

func (b *WaitBarrier) Start() WaitBarrierContext {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.wg == nil {
		b.wg = new(sync.WaitGroup)
	}
	b.wg.Add(1)
	return WaitBarrierContext{wg: b.wg}
}

func (b *WaitBarrier) Barrier() {
	b.BarrierWithUnlock(nil)
}

func (b *WaitBarrier) BarrierWithUnlock(u Unlocker) {
	b.lock.Lock()
	if b.wg == nil {
		b.lock.Unlock()
		if u != nil {
			u.Unlock()
		}
		return
	}
	wg := b.wg
	b.wg = new(sync.WaitGroup)
	barrierWg := b.wg
	barrierWg.Add(1)
	b.lock.Unlock()

	if u != nil {
		u.Unlock()
	}

	wg.Wait()
	barrierWg.Done()
}
