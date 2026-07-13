# KV Store basic version

Basic version of KV store building from scratch just using a hashmap

### Key Idea

1. Keep a hashmap with key,value pair
2. All the keys should fit in the RAM
3. We will append values to a binary file
4. In hashmap we will keep offset so we can seek and read from file
5. For update we will not update we will just append and update the offset in hashmap
6. While deleting we will keep a tombstone flag we will mark it `-1` while reading we will know it is deleted
7. Record type will keep Put,Delete

### Record Structure

record_type | record_length | key_length | value_length | key_bytes | value_bytes

Assume:

```
record_type = 1 byte
record_length = 4 bytes
key_length = 4 bytes
value_length = 4 bytes
```

Record types:

```
PUT = 1
DELETE = 2
```

### Tombstones

These will be used to know that record is deleted so once we have read it from the data file for example:

`1 | 24 | 4 | 7 | name | Pranjal`

Meaning:

```sh
record_type = 1 // PUT
record_length = 24 // total bytes in this record
key_length = 4 // "name"
value_length = 7 // "Pranjal"
key_bytes = name
value_bytes = Pranjal
```

### append-only log

### hash index in memory

### segment files

### compaction

### crash recovery

### SSTables / LSM trees when appropriate
