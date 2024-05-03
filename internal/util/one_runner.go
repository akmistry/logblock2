package util

import (
	"sync"
)

type OneRunner struct {
	f       func()
	running bool
	run     bool
	lock    sync.Mutex
	cond    *sync.Cond
}

func NewOneRunner(f func()) *OneRunner {
	r := &OneRunner{f: f}
	r.cond = sync.NewCond(&r.lock)
	return r
}

func (r *OneRunner) do() {
	for {
		r.lock.Lock()
		run := r.run
		if !run {
			r.running = false
			r.cond.Broadcast()
		}
		r.run = false
		r.lock.Unlock()
		if !run {
			return
		}

		r.f()
	}
}

func (r *OneRunner) Go() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.run = true
	if !r.running {
		r.running = true
		go r.do()
	}
}

func (r *OneRunner) Run() {
	doRun := false
	r.lock.Lock()
	r.run = true
	if !r.running {
		r.running = true
		doRun = true
	}
	r.lock.Unlock()

	if doRun {
		r.do()
	}
}

func (r *OneRunner) Running() bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.running
}

func (r *OneRunner) Wait() {
	r.lock.Lock()
	defer r.lock.Unlock()

	for r.running {
		r.cond.Wait()
	}
}
