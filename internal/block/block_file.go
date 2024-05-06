package block

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/akmistry/logblock2/internal/adaptor"
	"github.com/akmistry/logblock2/internal/blockmap"
	"github.com/akmistry/logblock2/internal/metadata"
	"github.com/akmistry/logblock2/internal/sparseblock"
	"github.com/akmistry/logblock2/internal/storage"
	"github.com/akmistry/logblock2/internal/t"
	"github.com/akmistry/logblock2/internal/tracker"
	"github.com/akmistry/logblock2/internal/util"
)

const (
	DefaultTargetTableSize     = 1024 * 1024 * 1024
	DefaultMinTableUtilisation = 0.5

	MaxTableSize = 64 * 1024 * 1024 * 1024
	MinTableSize = 1024 * 1024

	MinBlockSize = 512
	MaxBlockSize = 1024 * 1024

	MaxBlocks = (1 << 48)

	sparseblockExt = ".sparseblock"
)

type BlockOptions struct {
	BlobStore           storage.BlobStore
	LogStore            storage.LogStore
	BlockSize           int
	NumBlocks           int64
	TargetTableSize     int64
	MinTableUtilisation float64
	MaxWriteRate        float64

	MetadataStore metadata.MetadataStore
}

type BlockFile struct {
	opts       BlockOptions
	dataSource storage.BlobStore
	logSource  storage.LogStore
	numBlocks  int64
	blockSize  int

	nextSeq uint64
	seqLock sync.Mutex

	readerTracker *tracker.Reader
	writerTracker *tracker.Writer

	writeCompactRunner *util.OneRunner

	meta *metadata.Metadata
}

func OpenBlockFileWithOptions(opts BlockOptions) (*BlockFile, error) {
	startTime := time.Now()

	if opts.BlobStore == nil || opts.LogStore == nil {
		return nil, errors.New("BlobStore and LogStore must be non-nil")
	}
	if opts.BlockSize < MinBlockSize || opts.BlockSize > MaxBlockSize {
		return nil, fmt.Errorf("BlockSize must be between %d and %d",
			MinBlockSize, MaxBlockSize)
	}
	if opts.NumBlocks < 0 || opts.NumBlocks > MaxBlocks {
		return nil, errors.New("opts.NumBlocks must between 0 and (1<<48)")
	}
	util.SetDefaultIfZero(&opts.TargetTableSize, DefaultTargetTableSize)
	if opts.TargetTableSize < MinTableSize || opts.TargetTableSize > MaxTableSize {
		return nil, errors.New("TargetTableSize must be between 1M and 64G")
	}
	util.SetDefaultIfZero(&opts.MinTableUtilisation, DefaultMinTableUtilisation)
	if opts.MinTableUtilisation < 0.1 || opts.MinTableUtilisation > 0.9 {
		return nil, errors.New("MinTableUtilisation must be between 0.1 and 0.9")
	}

	bf := &BlockFile{
		opts:          opts,
		dataSource:    opts.BlobStore,
		logSource:     opts.LogStore,
		numBlocks:     opts.NumBlocks,
		blockSize:     opts.BlockSize,
		nextSeq:       1,
		readerTracker: tracker.NewReader(opts.BlockSize, opts.NumBlocks),
		writerTracker: tracker.NewWriter(opts.BlockSize),
	}
	bf.writeCompactRunner = util.NewOneRunner(bf.compactWriter)

	var err error
	bf.meta, err = metadata.LoadMetadata(opts.MetadataStore)
	if errors.Is(err, fs.ErrNotExist) {
		slog.Warn("Metadata not found, creating new device")
		bf.meta = metadata.NewMetadata_BlockStore(
			opts.MetadataStore, bf.blockSize, uint64(bf.numBlocks))
	} else if err != nil {
		return nil, fmt.Errorf("error loading metadata: %v", err)
	}

	dfs := bf.meta.ListDataFiles()
	for _, df := range dfs {
		name := df.Name()
		switch df.Type() {
		case metadata.DataFile_WRITE_LOG:
			f, err := bf.loadDataFile(df)
			if err != nil {
				return nil, err
			}
			bf.readerTracker.PushReader(f)
			bf.writerTracker.PushPending(f.(*WriteLog))
		case metadata.DataFile_SPARSE_TABLE:
			f, err := bf.loadDataFile(df)
			if err != nil {
				return nil, err
			}
			bf.readerTracker.PushReader(f)

			// Figure out the maximum sequence number so that we know what to call
			// the next compacted log file.
			parts := strings.Split(name, ".")
			seq, err := strconv.ParseUint(parts[0], 10, 64)
			if err != nil {
				slog.Error("Error parsing table sequence", "name", name, "error", err)
			} else if seq >= bf.nextSeq {
				bf.nextSeq = seq + 1
			}
		default:
			slog.Error("Unexpected file type", "name", name, "type", df.Type())
		}
	}

	slog.Info(fmt.Sprintf("Next table sequence number: %d\n", bf.nextSeq))
	slog.Debug(fmt.Sprintf("Metadata: \n%v\n", bf.meta))

	// Open new write log
	newWriteLog := bf.makeLogWriter()
	bf.meta.PushWriteLog(newWriteLog.Name())
	bf.meta.SaveNoError()

	bf.readerTracker.PushWriter(newWriteLog)
	bf.writerTracker.PushNewLog(newWriteLog)

	runtime.GC()

	slog.Info("Finished loading device", "loadTime", time.Since(startTime))

	return bf, nil
}

func (bf *BlockFile) loadDataFile(df metadata.DataFile) (t.BlockHoleReader, error) {
	name := df.Name()
	switch df.Type() {
	case metadata.DataFile_WRITE_LOG:
		logLoadStart := time.Now()
		lf, err := bf.logSource.Open(name)
		if err != nil {
			return nil, fmt.Errorf("error opening log file %s on load: %v", name, err)
		}
		wl, err := NewWriteLogFromReader(lf, lf.Size(), bf.blockSize, name)
		if err != nil {
			lf.Close()
			return nil, fmt.Errorf("error loading log file %s: %v", name, err)
		}
		slog.Info("Loaded log file", "name", name, "loadTime", time.Since(logLoadStart))

		return wl, nil
	case metadata.DataFile_SPARSE_TABLE:
		ltf, err := bf.openTableFile(name)
		if err != nil {
			slog.Error("Unable to load table", "name", name, "error", err)
		}
		return ltf, err
	}

	slog.Error("Unexpected file type", "name", name, "type", df.Type())
	return nil, fmt.Errorf("Unknown file type: %d", df.Type())
}

func (bf *BlockFile) openTableFile(name string) (*DataFile, error) {
	startTime := time.Now()
	f, err := bf.dataSource.Open(name)
	if err != nil {
		return nil, err
	}
	df, err := NewSparseBlockFromReader(f, bf.blockSize, bf.Size(), name)
	if err != nil {
		f.Close()
		return nil, err
	}
	slog.Info("Opened sparse file", "name", name, "loadTime", time.Since(startTime))
	return df, nil
}

func (bf *BlockFile) createFile() (storage.BlobWriter, string, error) {
	name := bf.generateNextTableName()
	f, err := bf.dataSource.Create(name)
	if err != nil {
		return nil, "", err
	}
	return f, name, nil
}

func (bf *BlockFile) makeLogWriter() *WriteLog {
	for {
		name := fmt.Sprintf("wal-%08d", rand.Intn(10000000))
		lf, err := bf.logSource.Create(name)
		if errors.Is(err, fs.ErrExist) {
			// Log already exists, try a different name
			continue
		} else if err != nil {
			panic(err)
		}
		return NewWriteLogFromWriter(lf, bf.blockSize, name)
	}
}

func (bf *BlockFile) generateNextTableName() string {
	bf.seqLock.Lock()
	seq := bf.nextSeq
	bf.nextSeq++
	bf.seqLock.Unlock()

	return strconv.FormatUint(seq, 10) + sparseblockExt
}

func (bf *BlockFile) compactToFile(r t.HoleReaderAt) (string, error) {
	startTime := time.Now()
	writeFile, name, err := bf.createFile()
	if err != nil {
		return "", err
	}
	slog.Debug("Creating table file", "name", name)

	buildStartTime := time.Now()
	err = sparseblock.BuildFromReader(writeFile, r, bf.blockSize, nil)
	buildTime := time.Now()
	if err != nil {
		slog.Error("Unable to build table", "name", name, "error", err)
		bf.dataSource.Remove(name)
		return "", err
	}
	err = writeFile.Close()
	closeTime := time.Now()
	if err != nil {
		slog.Error("Unable to close table", "name", name, "error", err)
		bf.dataSource.Remove(name)
		return "", err
	}

	log.Printf("Finish table write, total time: %v, "+
		"file create time: %v, "+
		"build time: %v, "+
		"close time: %v",
		time.Since(startTime),
		buildStartTime.Sub(startTime),
		buildTime.Sub(buildStartTime),
		closeTime.Sub(buildTime))

	return name, nil
}

func (bf *BlockFile) rotateNewWriteLog() {
	newWriteLog := bf.makeLogWriter()

	// Push the new log into the readers before swapping the writer, so that
	// we can have read-after-write consistency.
	bf.readerTracker.PushWriter(newWriteLog)

	// TODO: Save an index of the log somewhere to speed up loading if a
	// compaction doesn't take place, or is interrupted.

	// Save metadata so writes don't become lost if we crash right after
	// swapping the log.
	bf.meta.PushWriteLog(newWriteLog.Name())
	bf.meta.SaveNoError()

	oldWriter := bf.writerTracker.PushNewLog(newWriteLog)
	if oldWriter != nil {
		// Old writer is no longer being written to, so push it as an
		// immutable reader.
		bf.readerTracker.PushReader(oldWriter.(*WriteLog))
		bf.readerTracker.RemoveWriter(oldWriter.(*WriteLog))
	}
}

func (bf *BlockFile) compactPendingWriters() {
	// Since write log compaction is the only time utilisation can change,
	// co-compact low utilisation tables with the write log.
	cs := bf.readerTracker.StartCompaction()
	for cs == nil {
		slog.Warn("Compaction in progress, retry in 1 second...")
		time.Sleep(time.Second)
		cs = bf.readerTracker.StartCompaction()
	}
	defer cs.Done()

	startTime := time.Now()
	defer func() {
		slog.Info("Writer compaction done", "buildTime", time.Since(startTime))
	}()

	indexReader := blockmap.NewReader(bf.blockSize, bf.numBlocks)
	compactionReader := BuildCompactionReader(cs, bf.blockSize, bf.opts.MinTableUtilisation, bf.opts.TargetTableSize)
	if compactionReader != nil {
		indexReader.PushReader(compactionReader)
	}

	compactingWriters := bf.writerTracker.Pending()
	log.Printf("Pushing %d write logs for compaction", len(compactingWriters))
	addStart := time.Now()
	for _, w := range compactingWriters {
		indexReader.PushReader(w.(*WriteLog))
	}
	log.Printf("Writer add for compaction time %v", time.Since(addStart))

	compactionBlocks := indexReader.LiveBlocks()
	compactionBytes := compactionBlocks * int64(bf.blockSize)
	log.Printf("Total compaction bytes: %v", util.DetailedBytes(compactionBytes))

	// TODO: Split compaction into multiple tables
	//numTables := (compactionBytes / bf.opts.TargetTableSize) + 1
	//tableSize := compactionBytes / numTables

	compactStartTime := time.Now()
	compactedName, err := bf.compactToFile(adaptor.NewHoleReader(indexReader, bf.blockSize))
	if err != nil {
		// TODO: Re-try in the future instead of panicing
		panic(err)
	}
	openStartTime := time.Now()
	df, err := bf.openTableFile(compactedName)
	openTime := time.Now()
	if err != nil {
		bf.dataSource.Remove(compactedName)
		panic(err)
	}
	cs.PushCompactedReader(df)
	pushTime := time.Now()
	bf.writerTracker.RemovePending(compactingWriters)

	log.Printf("Finish writer compaction, total time: %v, "+
		"open time: %v, push time: %v",
		time.Since(compactStartTime),
		openTime.Sub(openStartTime),
		pushTime.Sub(openTime))
	// TODO: This could lead to inconsistent data file ordering
	bf.meta.CompactWriteLog(util.SliceLast(compactingWriters).Name(), compactedName)

	// This will also save the metadata
	bf.removeUnusedTables(cs)
	//bf.meta.SaveNoError()
}

func (bf *BlockFile) compactWriter() {
	written := bf.writerTracker.LogWritten()
	if written < bf.opts.TargetTableSize {
		return
	}
	log.Printf("Rotating write log with approx size %v", util.DetailedBytes(written))

	bf.rotateNewWriteLog()

	// Compact the old log file and any low-utilisation data files
	bf.compactPendingWriters()
}

func (bf *BlockFile) removeUnusedTables(cs *tracker.CompactionState) {
	liveBlocks := cs.LiveBlocks()

	type nameCloser interface {
		io.Closer
		Name() string
	}

	var unused []nameCloser
	for r, live := range liveBlocks {
		if live > 0 {
			continue
		}
		cs.RemoveUnusedReader(r)

		switch f := r.(type) {
		case *DataFile:
			bf.meta.RemoveSparseDataFile(f.Name())
			unused = append(unused, f)
		case *WriteLog:
			bf.meta.RemoveWriteLog(f.Name())
			unused = append(unused, f)
		default:
			slog.Error("Unrecognised unused reader type",
				"type", fmt.Sprintf("%T", r),
				"value", r)
		}
	}
	if len(unused) > 0 {
		bf.meta.SaveNoError()
	}

	for _, r := range unused {
		slog.Info("Removing unused file", "name", r.Name())
		err := r.Close()
		if err != nil {
			slog.Error("Error closing file", "name", r.Name(), "error", err)
		}

		switch f := r.(type) {
		case *DataFile:
			err = bf.dataSource.Remove(f.Name())
			if err != nil {
				slog.Error("Error deleting data file", "name", f.Name(), "error", err)
			}
		case *WriteLog:
			err = bf.logSource.Remove(f.Name())
			if err != nil {
				slog.Error("Error deleting log file", "name", f.Name(), "error", err)
			}
		}
	}

	// Data files may have been closed/removed, run a GC to try and free up some
	// memory.
	runtime.GC()
}

func (bf *BlockFile) BlockSize() int {
	return bf.blockSize
}

func (bf *BlockFile) Size() int64 {
	return int64(bf.blockSize) * bf.numBlocks
}

func (bf *BlockFile) Close() error {
	// Wait for any active compaction to finish.
	bf.writeCompactRunner.Wait()

	emptyLogs, err := bf.writerTracker.Close()
	if len(emptyLogs) > 0 {
		for _, l := range emptyLogs {
			bf.meta.RemoveWriteLog(l)
		}
		bf.meta.SaveNoError()
		for _, l := range emptyLogs {
			slog.Info("Removing empty log file", "name", l)
			err := bf.logSource.Remove(l)
			if err != nil {
				log.Printf("Error removing write log on close %s: %v", l, err)
			}
		}
	}

	return err
}

func (bf *BlockFile) ReadBlocks(buf []byte, index int64) (blocksRead int, err error) {
	return bf.readerTracker.ReadBlocks(buf, index)
}

func (bf *BlockFile) WriteBlocks(buf []byte, index int64) (blocksRead int, err error) {
	n, err := bf.writerTracker.WriteBlocks(buf, index)

	// This is a rough count of bytes written to the current log, and not used
	// for any actual decision making.
	// Note: This can be negative since it's incremented after the write, and
	// the writer could have already been swapped by the time this is done.
	written := bf.writerTracker.LogWritten()
	if written > bf.opts.TargetTableSize {
		bf.writeCompactRunner.Go()
	}

	return n, err
}

func (bf *BlockFile) Flush() error {
	startTime := time.Now()
	defer func() {
		slog.Info("Flush done", "flushTime", time.Since(startTime))
	}()

	return bf.writerTracker.Flush()
}
