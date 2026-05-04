# Nebula-KV

A Minimal Distributed KV Storage Engine with Raft Consensus and MVCC Support

## Overview

Nebula-KV is a minimal distributed key-value storage engine built from scratch in Go. It implements core components found in production-grade databases like TiKV, RocksDB, and etcd.

This project is designed for learning and demonstrating:

- How distributed consensus algorithms (Raft) work
- How LSM-Tree based storage engines operate
- How MVCC (Multi-Version Concurrency Control) is implemented
- How to build a complete database kernel

## Features

| Component | Description | Status |
|-----------|-------------|--------|
| MemTable | In-memory skip list for fast O(log n) reads/writes | Done |
| WAL | Write-Ahead Logging for crash recovery | Done |
| SSTable | Sorted String Table for persistent storage | Done |
| Engine | Storage engine that integrates all components | Done |
| Raft | Consensus algorithm for distributed replication | Next |
| MVCC | Multi-Version Concurrency Control | Planned |
| gRPC | Network protocol for remote access | Planned |

### Technical Highlights

- O(log n) Performance: Skip list provides efficient in-memory operations
- Crash Recovery: WAL ensures data durability
- Persistent Storage: SSTable provides disk-based storage
- Concurrency Safe: RWMutex for thread-safe operations
- Binary Protocol: Efficient serialization for storage

## Architecture

    +-----------------+     +-----------------+     +-----------------+
    |    MemTable     |     |       WAL       |     |    SSTable      |
    |   (Skip List)   |     |  (Write-Ahead)  |     |  (Sorted Disk)  |
    |    In-Memory    |     |    Persistent   |     |    Persistent   |
    +-----------------+     +-----------------+     +-----------------+

### Data Flow

Write Path:

    Client Request -> WAL (append) -> MemTable (insert) -> Return Success

Read Path:

    Client Request -> MemTable -> SSTables (newest to oldest) -> Return Value

Flush Process:

    MemTable (full) -> Build SSTable -> Clear WAL -> Add to SSTable List

### Component Details

#### 1. MemTable (Skip List)

The MemTable is an in-memory data structure that stores recent writes.

Skip List Structure:

    Level 3:  1 --------------- 9 ----------- 15
    Level 2:  1 --------- 5 --------- 9 ----------- 15
    Level 1:  1 -- 3 -- 5 -- 7 -- 9 -- 11 -- 13 -- 15

Characteristics:

- O(log n) average case for search, insert, and delete
- Probabilistic data structure with O(n) space
- Supports concurrent operations with RWMutex

#### 2. WAL (Write-Ahead Log)

The WAL ensures data durability by logging all operations before applying them.

WAL Entry Format:

    +----------+----------+---------+---------+
    | Checksum | Key Len  | Value   | Key +   |
    | (4 bytes)| (4 bytes)| Len     | Value   |
    |          |          | (4 bytes)|         |
    +----------+----------+---------+---------+

Characteristics:

- Append-only writes for high throughput
- CRC32 checksum for data integrity
- Sequential I/O for performance

#### 3. SSTable (Sorted String Table)

SSTables are immutable, sorted files stored on disk.

SSTable File Format:

    +-----------------------------------------+
    |           [Data Block]                  |
    |  key1 -> value1                         |
    |  key2 -> value2                         |
    |  ...                                    |
    +-----------------------------------------+
    |           [Index Block]                 |
    |  key1 -> offset:0, size:100             |
    |  key2 -> offset:100, size:150           |
    |  ...                                    |
    +-----------------------------------------+
    |           [Footer]                      |
    |  index_offset: 500                      |
    +-----------------------------------------+

Characteristics:

- Immutable once written
- Binary search on index for O(log n) lookup
- Supports range queries

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Git

### Installation

    git clone https://github.com/t88173871-lang/nebula-kv.git
    cd nebula-kv
    go build -o nebula-cli ./cmd/nebula-cli/

### Usage

Interactive CLI Mode:

    go run ./cmd/nebula-cli/main.go

Example Session:

    Nebula-KV CLI Client
    ========================
    Commands: put <key> <value>, get <key>, delete <key>, stats, exit

    Engine initialized. Ready to accept commands!

    nebula> put name Alice
    OK - Set 'name' = 'Alice'

    nebula> put age 25
    OK - Set 'age' = '25'

    nebula> get name
    "Alice"

    nebula> get age
    "25"

    nebula> stats
    Engine Statistics:
    --------------------
      memtable_size  : 2
      sstable_count  : 0
      data_dir       : ./data

    nebula> delete age
    OK - Deleted 'age'

    nebula> get age
    (nil) - Key 'age' not found

    nebula> exit
    Goodbye!

Command-Line Mode:

    go run ./cmd/nebula-kv-cli/main.go put user:1001 "{'name':'Alice','age':25}"
    go run ./cmd/nebula-kv-cli/main.go get user:1001
    go run ./cmd/nebula-kv-cli/main.go delete user:1001
    go run ./cmd/nebula-kv-cli/main.go stats

## Project Structure

    nebula-kv/
    ├── cmd/
    │   ├── nebula-kv/              # Main server (planned)
    │   ├── nebula-cli/             # Interactive CLI client
    │   │   └── main.go
    │   └── nebula-kv-cli/          # Command-line client
    │       └── main.go
    ├── internal/
    │   ├── memtable/               # Skip list implementation
    │   │   ├── skiplist.go
    │   │   └── skiplist_test.go
    │   ├── wal/                    # Write-Ahead Logging
    │   │   ├── wal.go
    │   │   └── wal_test.go
    │   ├── sstable/                # Sorted String Table
    │   │   ├── sstable.go
    │   │   └── sstable_test.go
    │   ├── engine/                 # Storage engine
    │   │   ├── engine.go
    │   │   └── engine_test.go
    │   ├── raft/                   # Raft consensus (planned)
    │   ├── mvcc/                   # MVCC transactions (planned)
    │   └── server/                 # gRPC server (planned)
    ├── pkg/                        # Exportable packages
    ├── data/                       # Runtime data (gitignored)
    ├── go.mod                      # Go module definition
    ├── go.sum                      # Go module checksums
    ├── .gitignore                  # Git ignore rules
    └── README.md                   # This file

## Testing

Run All Tests:

    go test ./... -v
    go test ./... -cover

Run Specific Module Tests:

    go test ./internal/memtable/... -v
    go test ./internal/wal/... -v
    go test ./internal/sstable/... -v
    go test ./internal/engine/... -v

### Test Coverage

| Module | Tests | Status |
|--------|-------|--------|
| MemTable | Basic operations, ForEach traversal, Concurrency | Pass |
| WAL | Basic operations, Recovery, Truncate | Pass |
| SSTable | Basic operations, ForEach traversal, Deleted markers | Pass |
| Engine | Basic operations, Recovery, Flush | Pass |

## Performance

### Time Complexity

| Operation | MemTable | SSTable | WAL |
|-----------|----------|---------|-----|
| Put | O(log n) | N/A | O(1) |
| Get | O(log n) | O(log n) | N/A |
| Delete | O(log n) | N/A | O(1) |

### Space Complexity

| Component | Space |
|-----------|-------|
| MemTable | O(n) |
| WAL | O(operations) |
| SSTable | O(data) |

## Roadmap

### Phase 1: Single-Node Storage Engine (Done)

- MemTable with Skip List
- Write-Ahead Logging (WAL)
- SSTable with binary search
- Storage engine integration
- CLI client for testing

### Phase 2: Distributed Consensus (Next)

- Raft leader election
- Log replication
- Membership changes
- Snapshot support
- Network transport layer

### Phase 3: Transactions & MVCC

- Timestamp-based MVCC
- Snapshot isolation
- Conflict detection
- Optimistic transactions

### Phase 4: Network & API

- gRPC server
- Binary protocol
- Batch operations
- Range queries
- Client SDK

### Phase 5: Optimization

- Leveled compaction
- Bloom filters
- Block cache
- Compression
- Performance benchmarks

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### How to Contribute

1. Fork the repository
2. Create your feature branch

        git checkout -b feature/amazing-feature

3. Commit your changes

        git commit -m 'feat: add amazing feature'

4. Push to the branch

        git push origin feature/amazing-feature

5. Open a Pull Request

### Commit Convention

We follow the Conventional Commits specification:

- feat: New feature
- fix: Bug fix
- docs: Documentation
- style: Code style changes
- refactor: Code refactoring
- test: Adding tests
- chore: Maintenance

## License

This project is licensed under the MIT License.

    MIT License

    Copyright (c) 2024 t88173871-lang

    Permission is hereby granted, free of charge, to any person obtaining a copy
    of this software and associated documentation files (the "Software"), to deal
    in the Software without restriction, including without limitation the rights
    to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
    copies of the Software, and to permit persons to whom the Software is
    furnished to do so, subject to the following conditions:

    The above copyright notice and this permission notice shall be included in all
    copies or substantial portions of the Software.

    THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
    IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
    FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
    AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
    LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
    OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
    SOFTWARE.

## Acknowledgments

This project is inspired by and built upon the work of:

- TiKV - Distributed transactional KV database
- LevelDB - Fast key-value storage library
- RocksDB - Persistent key-value store
- etcd - Distributed reliable key-value store
- Raft Paper - The Raft consensus algorithm

Special thanks to:

- The Go team for the excellent programming language
- The open-source community for inspiration and tools

## Contact

- Author: t88173871-lang
- GitHub: t88173871-lang
- Repository: nebula-kv
