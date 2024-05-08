# Logblock: An LSM-based block device

Logblock is a software block device built on the concept of [log-structured
merge-trees](https://en.wikipedia.org/wiki/Log-structured_merge-tree) (LSMs).

Logblock is inspired by how the Flash Translation Layer (FTLs) of an SSD works.
But instead of NAND devices, we have blob storage systems, which are
conceptually similar at the API level. By pushing the FTL up the stack, we
can build block devices on top of abstract distributed storage systems,
decoupling the block device from the underlying storage hardware.
This is more CPU/memory instensive than simply mapping block ranges on a
storage device to a logical block device, which is largely how traditional
distributed block device are implemented. But decoupling from the storage
hardware create a large amount of flexibility in terms of data allocation,
replication, encoding, etc.

## How to use

Currently, logblock is very basic and only supports storing data on the
local filesystem. The logblock binary is located in `cmd/logblock` and can be
run as:
```
% ./logblock -size 64G /dev/nbd0 /path/to/data/storage/directory
% sudo mke2fs /dev/nbd0
% sudo mount /dev/nbd0 /mnt/test
```

## Overview

Conceptually, a block device can be thought of as a key-value store (a very
limited one). The key is a fixed 8-byte integer, which is the block index
(or LBA), and the value is simply the block data. Therefore, one can trivially
build a block device on top of their favourite key-value database.

This is not particularly useful for building a locally-attached block device
(maybe except for simulating a very large device for testing). However, one can
use a distributed KV-store to build scalable, fault-tolerent block devices for
VMs and other applications.

Logblock goes one step further, and instead of building on top of a database,
it leverages the properties of a block device (i.e. fixed-size keys and values)
to build a more efficient solution.

As an LSM, writes are written to a write-ahead log. Once the log reaches a
certain size, it is "compacted" into a sparse block format. A block mapping
keeps track of which file contains which block. The compaction step also frees
up space which has been overwritten by more recently written blocks.

## QNAs (Questions Nobody Asked)

### Why is it called "logblock2"?

Because it's a "Log"-structured "Block" device. Also, naming is hard.

### No, why is it called logblock_2_?

Because I have an original logblock git repo, which is >5 years old. It has
multiple iterations of the write-ahead log and sparse block format, various
block mapping data structures, a "graveyard" of unused code, and a git history
which is starting to look like https://xkcd.com/1296/. It was probably best to
start a new repo as I open-source this project.

### How is this better than existing-distributed-block-device?

Honestly, it isn't. For one, this isn't a complete project, but rather a
technical demo. It has too many missing features to be useful, not the least
of which is integration with a blob/log storage backend.

## TODO
- TRIM support
- Re-designed metadata model
- Storage backends (i.e. S3, HDFS, etc)
- Performance improvements
- More tests
- Alternate compaction strategies

## LICENSE

Logblock is released under a BSD 3-Clause License.
