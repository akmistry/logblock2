package tracker

import (
	"errors"
	"io"
	"log/slog"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/akmistry/logblock2/internal/util"
)

var (
	ErrNoWriter = errors.New("no writer")
)

type TrackedWriter interface {
	WriteBlocks(buf []byte, index int64) (blocksWritten int, err error)
	io.Closer
	Flush() error
	CloseWriter() error

	Name() string
	Written() int64
}

type WriterTracker struct {
	blockSize        int
	activeWriter     TrackedWriter
	activeWriterLock sync.Mutex
	activeWriterWb   util.WaitBarrier
	writeCount       atomic.Int64

	pendingWriters     []TrackedWriter
	pendingWritersLock sync.Mutex
	pendingWritersWb   util.WaitBarrier
}

func NewWriterTracker(blockSize int) *WriterTracker {
	return &WriterTracker{
		blockSize: blockSize,
	}
}

func (m *WriterTracker) PushPending(wl TrackedWriter) {
	m.pendingWritersLock.Lock()
	m.pendingWriters = append(m.pendingWriters, wl)
	m.pendingWritersLock.Unlock()
}

func (m *WriterTracker) RemovePending(p []TrackedWriter) {
	m.pendingWritersLock.Lock()
	defer m.pendingWritersWb.BarrierWithUnlock(&m.pendingWritersLock)

	for _, rp := range p {
		for j := range m.pendingWriters {
			if rp == m.pendingWriters[j] {
				m.pendingWriters = slices.Delete(m.pendingWriters, j, j+1)
				break
			}
		}
	}
}

func (m *WriterTracker) Pending() []TrackedWriter {
	m.pendingWritersLock.Lock()
	defer m.pendingWritersLock.Unlock()
	return slices.Clone(m.pendingWriters)
}

func (m *WriterTracker) PushNewLog(wl TrackedWriter) TrackedWriter {
	// Make the current and pendings writers consistent from the point of view
	// of any active Flush().
	m.pendingWritersLock.Lock()
	m.activeWriterLock.Lock()
	oldWriter := m.activeWriter
	m.activeWriter = wl
	if oldWriter == nil {
		m.activeWriterLock.Unlock()
		m.pendingWritersLock.Unlock()
		return nil
	}
	m.pendingWriters = append(m.pendingWriters, oldWriter)

	done := m.pendingWritersWb.Start()
	m.pendingWritersLock.Unlock()

	m.activeWriterWb.BarrierWithUnlock(&m.activeWriterLock)
	// At this point, there are no outstanding or new writes to the old log.
	m.writeCount.Store(0)

	// Proactively flush the old writer
	go func() {
		// Close the old writer to write out the log index, in case we crash
		// during compaction.
		err := oldWriter.CloseWriter()
		if err != nil {
			slog.Error("Unable to close log writer", "error", err)
		}
		done.Done()
	}()

	return oldWriter
}

func (m *WriterTracker) WriteBlocks(buf []byte, index int64) (blocksRead int, err error) {
	m.activeWriterLock.Lock()
	writer := m.activeWriter
	defer m.activeWriterWb.Start().Done()
	m.activeWriterLock.Unlock()

	if writer == nil {
		return 0, ErrNoWriter
	}

	n, err := writer.WriteBlocks(buf, index)
	if n > 0 {
		m.writeCount.Add(int64(n * m.blockSize))
	}
	return n, err
}

func (m *WriterTracker) LogWritten() int64 {
	return m.writeCount.Load()
}

func (m *WriterTracker) Flush() error {
	startTime := time.Now()
	defer func() {
		slog.Info("Flush done", "flushTime", time.Since(startTime))
	}()

	m.pendingWritersLock.Lock()
	flushWriters := slices.Clone(m.pendingWriters)
	defer m.pendingWritersWb.Start().Done()
	m.activeWriterLock.Lock()
	if m.activeWriter != nil {
		flushWriters = append(flushWriters, m.activeWriter)
		defer m.activeWriterWb.Start().Done()
	}
	m.activeWriterLock.Unlock()
	m.pendingWritersLock.Unlock()

	// Flush in parallel
	var wg sync.WaitGroup
	wg.Add(len(flushWriters))
	for _, w := range flushWriters {
		go func(wt TrackedWriter) {
			err := wt.Flush()
			if err != nil {
				slog.Error("Unable to flush writer", "name", wt.Name(), "error", err)
			}
			wg.Done()
		}(w)
	}
	wg.Wait()

	return nil
}

func (m *WriterTracker) Close() ([]string, error) {
	m.pendingWritersLock.Lock()
	m.activeWriterLock.Lock()
	oldWriter := m.activeWriter
	m.activeWriter = nil
	if oldWriter != nil {
		m.writeCount.Store(0)
		m.pendingWriters = append(m.pendingWriters, oldWriter)
	}
	// Wait for any active writes.
	m.activeWriterWb.BarrierWithUnlock(&m.activeWriterLock)
	// Wait for any flushes in progress.
	m.pendingWritersWb.BarrierWithUnlock(&m.pendingWritersLock)

	m.pendingWritersLock.Lock()
	defer m.pendingWritersLock.Unlock()

	var emptyLogs []string
	for _, w := range m.pendingWriters {
		name := w.Name()
		err := w.Close()
		if err != nil {
			slog.Warn("Unable to close write log", "name", name, "error", err)
		}
		if w.Written() == 0 {
			emptyLogs = append(emptyLogs, name)
		}
	}

	return emptyLogs, nil
}
