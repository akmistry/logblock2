package block

import (
	"log"
	"log/slog"
	"sort"
	"time"

	"github.com/akmistry/logblock2/internal/t"
	"github.com/akmistry/logblock2/internal/tracker"
	"github.com/akmistry/logblock2/internal/util"
)

type namer interface {
	Name() string
}

func BuildCompactionReader(cs *tracker.CompactionState, blockSize int, minUtilisation float64, maxCompactionBytes int64) t.BlockHoleReader {
	startTime := time.Now()
	defer func() {
		slog.Info("Building compaction reader done", "duration", time.Since(startTime))
	}()

	// Look for low utilisation tables
	liveBlocks := cs.LiveBlocks()

	type compactInfo struct {
		df          *DataFile
		utilisation float64
		liveBytes   int64
	}
	var cis []compactInfo
	totalCompactionBytes := int64(0)
	for r, blocks := range liveBlocks {
		live := blocks * int64(blockSize)
		df, ok := r.(*DataFile)
		if !ok {
			if nm, ok := r.(namer); ok {
				log.Printf("Skipping non-table data file, name: %s", nm.Name())
			} else {
				log.Print("Skipping non-table data file")
			}
			continue
		}
		utilisation := float64(live) / float64(df.DataSize())
		if utilisation > minUtilisation {
			log.Printf("Skipping %s (%v, %0.1f%% utilisation)", df.Name(), util.DetailedBytes(live), 100*utilisation)
			continue
		} else if blocks == 0 {
			log.Printf("Skipping empty %s, will be removed later", df.Name())
			continue
		}

		log.Printf("Adding %s (%v, %0.1f%% utilisation) to potential compaction list", df.Name(), util.DetailedBytes(live), 100*utilisation)

		// TODO: Limit the amount of data being compacted with the write logs.
		cis = append(cis, compactInfo{
			df:          df,
			utilisation: utilisation,
			liveBytes:   live,
		})
		totalCompactionBytes += live
	}
	log.Printf("Total table compaction bytes: %v", util.DetailedBytes(totalCompactionBytes))
	if len(cis) == 0 {
		return nil
	}

	sort.Slice(cis, func(i, j int) bool {
		return cis[i].liveBytes < cis[j].liveBytes
	})

	var rs []t.BlockHoleReader
	chosenTablesBytes := int64(0)
	for _, info := range cis {
		chosenTablesBytes += info.liveBytes
		rs = append(rs, info.df)
		log.Printf("Chose %s (%v) to compact",
			info.df.Name(), util.DetailedBytes(info.liveBytes))

		if chosenTablesBytes > maxCompactionBytes {
			break
		}
	}

	return cs.LiveBytesReader(rs)
}
