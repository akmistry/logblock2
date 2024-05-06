package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/akmistry/go-nbd"

	"github.com/akmistry/logblock2/internal/app/logblock"
	"github.com/akmistry/logblock2/internal/block"
	"github.com/akmistry/logblock2/internal/storage/local"
	"github.com/akmistry/logblock2/internal/adaptor"
)

var (
	sizeFlag    = flag.String("size", "", "Device size")
	verboseFlag = flag.Bool("verbose", false, "Verbose logging")

	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

const (
	// TODO: Raise artificial limit.
	maxDeviceSize = 16 * (1 << 40)

	blockSize       = 4096
	targetTableSize = 1024 * 1024 * 1024
)

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		log.Print("Usage: logblock <NDB_DEVICE> <DATA_DIR>")
		os.Exit(1)
	}

	nbdDev := flag.Arg(0)
	dataDir := flag.Arg(1)

	if *verboseFlag {
		slog.SetDefault(slog.New(slog.NewTextHandler(
			os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

	nbdUseNetlink := false
	nbdIndex, err := logblock.ParseNbdIndex(nbdDev)
	if err == nil {
		nbdUseNetlink = true
		log.Print("Using Netlink NBD interface")
	} else {
		log.Print("Using /dev/nbd* NBD interface")
	}

	var deviceSize uint64
	if *sizeFlag != "" {
		deviceSize, err = logblock.ParseSizeString(*sizeFlag)
		if err != nil {
			log.Printf("Invalid size flag: %s", *sizeFlag)
			os.Exit(1)
		}
	}

	if deviceSize > 0 && deviceSize%blockSize != 0 {
		log.Printf("Device size %s must be a multiple of block size %d",
			deviceSize, blockSize)
		os.Exit(1)
	} else if deviceSize > maxDeviceSize {
		log.Printf("Device size %s is too big (max 16T)", deviceSize)
		os.Exit(1)
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	go func() {
		log.Println("http.ListenAndServe: ", http.ListenAndServe("localhost:6060", nil))
	}()

  blobDir := filepath.Join(dataDir, "blobs")
  walDir := filepath.Join(dataDir, "wal")

	blobSource, err := local.NewBlobSource(blobDir)
	if err != nil {
		panic(err)
	}
	metaBs := block.NewBlobMetadataStore(blobSource)

	logSource, err := local.NewLogSource(walDir)
	if err != nil {
		panic(err)
	}

  blockOpts := block.BlockOptions{
    BlobSource:          blobSource,
    LogSource:           logSource,
    BlockSize:           blockSize,
    NumBlocks:          int64(deviceSize / blockSize),
    TargetTableSize:     targetTableSize,
    MinTableUtilisation: 0.5,
    MetadataStore:       metaBs,
  }

  bf, err := block.OpenBlockFileWithOptions(blockOpts)
  if err != nil {
    panic(err)
  }
  defer bf.Close()

		blockDev := adaptor.NewReadWriter(bf, blockSize)

	nbdOpts := nbd.BlockDeviceOptions{
		BlockSize:     blockSize,
		ConcurrentOps: 4,
	}
	var serv *nbd.NbdServer
	if nbdUseNetlink {
		serv, err = nbd.NewServerWithNetlink(nbdIndex, blockDev, int64(deviceSize), nbdOpts)
	} else {
		serv, err = nbd.NewServer(nbdDev, blockDev, int64(deviceSize), nbdOpts)
	}
	if err != nil {
		log.Println("Error creating NBD", err)
		os.Exit(1)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		<-ch
		log.Println("Shutting down after ^C. Will force after 10 seconds.")
		fin := make(chan bool)
		go func() {
			serv.Disconnect()
			close(fin)
		}()
		select {
		case <-fin:
		case <-time.After(10 * time.Second):
			log.Println("Force shutting down.")
			os.Exit(1)
		}
	}()

	err = serv.Run()
	if err != nil {
		log.Println("NBD run error: ", err)
		serv.Disconnect()
	}
}
