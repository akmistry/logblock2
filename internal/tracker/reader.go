package tracker

import (
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/akmistry/logblock2/internal/blockmap"
	"github.com/akmistry/logblock2/internal/readers"
	"github.com/akmistry/logblock2/internal/t"
	"github.com/akmistry/logblock2/internal/util"
)

type CompactionState struct {
	m *Reader
}

type Reader struct {
	blockSize int
	numBlocks int64

	// Main stack of readers, including any active writer.
	readerStack *readers.StackedBlockReader
	// Barrier for active readers
	wb util.WaitBarrier

	// Primary list of readers. This list is frozen when a compaction is started.
	// When compaction ends, the pending list is merged into this one.
	primaryIndex *blockmap.Reader

	// Readers pending inclusion into the reader list. Usually empty, except
	// when a (non-writer) compaction is in progress.
	pendingReaders []t.BlockHoleReader

	// Writers, including the currently active writer and any logs pending
	// compaction.
	writers []t.BlockHoleReader

	// Active compaction state. nil when no compaction is in progress.
	compactionState *CompactionState

	lock sync.Mutex
	cond *sync.Cond
}

func NewReader(blockSize int, numBlocks int64) *Reader {
	m := &Reader{
		blockSize:    blockSize,
		numBlocks:    numBlocks,
		primaryIndex: blockmap.NewReader(blockSize, numBlocks),
	}
	m.cond = sync.NewCond(&m.lock)
	m.buildReaderStack()
	go m.readerMergeLoop()
	return m
}

func (m *Reader) readerMergeLoop() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for {
		// Wait for any active compaction to finish.
		for len(m.pendingReaders) == 0 || m.compactionState != nil {
			m.cond.Wait()
		}

		// Make a copy of the currently pending readers.
		// Keep the readers in the pending list until after it is indexed, in
		// case new readers are added and the reader stack needs to be re-created.
		rs := slices.Clone(m.pendingReaders)

		// Even though the index is being added to here, this should be safe
		// since IndexedReader is internally locked, and any new ranges added
		// won't be referenced since they're served by the stack before reaching
		// primaryIndex.
		m.lock.Unlock()
		startTime := time.Now()
		// Pending readers are ordered most recent first, but we need to index
		// oldest first.
		slices.Reverse(rs)
		for _, br := range rs {
			m.primaryIndex.PushReader(br)
		}
		slog.Info("Reader: reader merge finished", "mergeTime", time.Since(startTime))
		m.lock.Lock()

		// Removed merged readers
		m.pendingReaders = slices.DeleteFunc(m.pendingReaders, func(e t.BlockHoleReader) bool {
			return slices.Contains(rs, e)
		})
		if len(m.pendingReaders) == 0 {
			// Signal any waiting compaction
			m.cond.Broadcast()
		}
		m.buildReaderStack()
		// This will release m.lock and re-acquire it when finished.
		m.waitActiveReaders()
	}
}

func (m *Reader) buildReaderStack() {
	var rs []t.BlockHoleReader
	if len(m.writers) > 0 {
		rs = append(rs, m.writers...)
	}
	if len(m.pendingReaders) > 0 {
		rs = append(rs, m.pendingReaders...)
	}
	rs = append(rs, m.primaryIndex)
	m.readerStack = readers.NewStackedBlockReader(rs, m.blockSize, m.numBlocks)
}

func (m *Reader) waitActiveReaders() {
	m.wb.BarrierWithUnlock(&m.lock)
	m.lock.Lock()
}

// TODO: Rename lots of things. "writers" should really be "mutable reader",
// and "reader" is really "immutable reader". The distiction is whether the
// reader contents can change (i.e. a write log), which determines whether
// we can add it to the block map.
func (m *Reader) PushWriter(w t.BlockHoleReader) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.writers = slices.Insert(m.writers, 0, w)
	m.buildReaderStack()
}

func (m *Reader) RemoveWriter(w t.BlockHoleReader) {
	m.lock.Lock()
	defer m.lock.Unlock()

	removed := false
	for i := range m.writers {
		if m.writers[i] == w {
			m.writers = slices.Delete(m.writers, i, i+1)
			removed = true
			break
		}
	}
	if !removed {
		slog.Error("Reader.RemoveWriter: writer not found", "writer", w)
		return
	}
	m.buildReaderStack()
	// The removed writer might be being read, so wait for any active readers to
	// finish.
	m.waitActiveReaders()
}

func (m *Reader) PushReader(r t.BlockHoleReader) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.pendingReaders = slices.Insert(m.pendingReaders, 0, r)
	m.buildReaderStack()
	m.cond.Broadcast()
}

func (m *Reader) ReadBlocks(p []byte, off int64) (int, error) {
	// Maintain a stack of active readers. This prevents readers and maintainence
	// operations (i.e. compactions) from blocking each other.
	m.lock.Lock()
	r := m.readerStack
	defer m.wb.Start().Done()
	m.lock.Unlock()

	return r.ReadBlocks(p, off)
}

func (c *CompactionState) LiveBlocks() map[t.BlockHoleReader]int64 {
	return c.m.primaryIndex.GetLiveBlocks()
}

func (c *CompactionState) LiveBytesReader(rs []t.BlockHoleReader) *blockmap.Reader {
	return c.m.primaryIndex.ReaderForLiveBytes(rs)
}

func (c *CompactionState) ListReaders() []t.BlockHoleReader {
	return c.m.primaryIndex.ListReaders()
}

func (c *CompactionState) PushCompactedReader(r t.BlockHoleReader) {
	startTime := time.Now()
	c.m.primaryIndex.PushReader(r)
	slog.Debug("CompactionState: PushCompactedReader", "pushTime", time.Since(startTime))
}

func (c *CompactionState) RemoveUnusedReader(r t.BlockHoleReader) {
	// Need to wait for any pending readers which might be reading the
	// reader being removed.
	c.m.wb.Barrier()

	c.m.primaryIndex.RemoveTable(r)
}

func (c *CompactionState) Reader() t.BlockHoleReader {
	return c.m.primaryIndex
}

func (c *CompactionState) Done() {
	c.m.lock.Lock()
	c.m.compactionState = nil
	// Signal any pending reader merges
	c.m.cond.Broadcast()
	c.m.lock.Unlock()

	c.m = nil
}

func (m *Reader) StartCompaction() *CompactionState {
	m.lock.Lock()
	defer m.lock.Unlock()

	if m.compactionState != nil {
		// Compaction in progress. Don't allow another one.
		return nil
	}

	for len(m.pendingReaders) > 0 {
		m.cond.Wait()
		if m.compactionState != nil {
			return nil
		}
	}

	m.compactionState = &CompactionState{
		m: m,
	}
	return m.compactionState
}
