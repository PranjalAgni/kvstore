# kvstore

A tiny persistent key-value store, built by hand while working through
**_Designing Data-Intensive Applications_ (DDIA)** by Martin Kleppmann.

This is a **learning project**. The goal is not to ship a database - it is to
practice the ideas from DDIA Chapter 3 (_Storage and Retrieval_) hands-on, so
they actually stick. Every design choice here maps to something in the book.
If a decision looks over-simple, that is on purpose: the point is to understand
one concept at a time, then layer the next one on top.

Written in Go, partly to learn idiomatic Go along the way.

## The idea: a Bitcask-style store

This follows the **Bitcask** model that DDIA uses to introduce log-structured
storage:

- **Append-only log on disk.** Every write (put or delete) is appended to the
  end of a single file. We never overwrite or edit existing bytes. The file is
  the durable, immutable history of everything that ever happened.
- **In-memory hash index (the "keydir").** A `map[string]int64` maps each key
  to the **byte offset** of its most recent record in the log. Reads are: look
  up the offset in memory, then `ReadAt` that spot in the file.

Two data structures, two different rules:

| | On-disk log (`data.bin`) | In-memory index (`hashMap`) |
|---|---|---|
| **Mutable?** | No - append-only, immutable | Yes - freely updated in RAM |
| **A put does** | append a new record | point the key at the new offset |
| **A delete does** | append a *tombstone* record | remove the key from the map |
| **Purpose** | durable source of truth | fast "where is the latest value?" index |

This design is tuned for a **read-heavy workload (~3:1 reads:writes)**: reads
are a single in-memory map lookup plus one positioned disk read, and hot keys
are served straight from the OS page cache.

## On-disk record format

Each record is length-prefixed so variable-length strings can be parsed back
out. All integers are **big-endian**.

```
+--------+-------------+----------+------------+-----------+-------------+
| op     | record_len  | key_len  | value_len  | key bytes | value bytes |
| 1 byte | 4 bytes     | 4 bytes  | 4 bytes    | variable  | variable    |
+--------+-------------+----------+------------+-----------+-------------+
 \___________________ 13-byte fixed header ___________________/
```

- `op` - `0` = put, `1` = delete (tombstone).
- `record_len` - total record size including the header. Lets a reader skip a
  whole record without parsing it (useful for replay and future compaction).
- `key_len` / `value_len` - byte lengths of the two strings (a delete tombstone
  has `value_len = 0`).

Example - `Set("go", "1.0")`:

```
00            op = put
00 00 00 12   record_len = 18  (13 header + 2 key + 3 value)
00 00 00 02   key_len = 2
00 00 00 03   value_len = 3
67 6f         "go"
31 2e 30      "1.0"
```

Inspect a real file with `xxd data.bin`.

## Durability

The write path is: `bufio.Writer` (app buffer) → `Flush()` (OS page cache) →
`Sync()`/fsync (physical disk). Right now every write does all three, so a
record is on physical disk before `Set` returns. This is the safest and slowest
setting - a deliberate starting point. Relaxing it (group commit / timed sync)
is a later experiment.

## What works today

- `Set`, `Get`, `Delete` in a single run.
- Append-only binary log with a full encode/decode path.
- In-memory keydir with byte offsets.
- Per-write fsync durability.

## Roadmap (the DDIA learning path)

- [ ] **`replay()` on startup.** Walk the log start→end and rebuild the keydir:
      a put sets the key, a tombstone `delete()`s it - "last write wins" as a
      fold over history. This is what makes the store survive restarts, and it
      is why delete writes a tombstone instead of erasing anything.
- [ ] **Fix delete's in-memory side.** A delete should `delete(hashMap, key)`,
      not point the key at the tombstone - the log keeps the tombstone, the
      index drops the key.
- [ ] **`NewKVStore` constructor** that opens the file, runs `replay()`, and
      sets `writeOffset` from the rebuilt state (replacing the current
      `f.Stat()` stopgap).
- [ ] **CRC32 checksum per record** - detect torn/corrupt records on replay.
- [ ] **Compaction** - garbage-collect superseded values and tombstones by
      rewriting the log into a fresh file.
- [ ] Clean up leftover learning scaffolding in `main.go`.

## Running

```sh
go run .
```

Writes/reads against `data.bin` in the working directory.

## Reference

- Kleppmann, _Designing Data-Intensive Applications_, Ch. 3 - "Storage and
  Retrieval" (hash indexes, SSTables, LSM-trees).
- Bitcask - the log + in-memory-hash-index design this project imitates.
