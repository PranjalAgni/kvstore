package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
)

const (
	DefaultPath      = "data.bin"
	HeaderSize       = 13
	opPut       byte = 0
	opDelete    byte = 1
)

type KVStore struct {
	hashMap     map[string]int64
	f           *os.File
	w           *bufio.Writer
	writeOffset int64
}

func print(kv map[string]string) {
	for k, v := range kv {
		fmt.Println(k, v)
	}
}

func encode(op byte, key, value string) []byte {
	var buf bytes.Buffer
	buf.WriteByte(op)
	binary.Write(&buf, binary.BigEndian, uint32(HeaderSize+len(key)+len(value)))
	binary.Write(&buf, binary.BigEndian, uint32(len(key)))
	binary.Write(&buf, binary.BigEndian, uint32(len(value)))
	buf.WriteString(key)
	buf.WriteString(value)
	return buf.Bytes()
}

func (kv *KVStore) append(op byte, key, value string) (int64, error) {
	offset := kv.writeOffset
	record := encode(op, key, value)
	if _, err := kv.w.Write(record); err != nil {
		return 0, err
	}

	// flush() just pushes data to OS level page cache
	// we will need to use fsync() to write data to actual disk
	if err := kv.w.Flush(); err != nil {
		return 0, err
	}

	// fsync() every write this is expensive because this makes program
	// pause on the hardware(disk) to tell data is persisted or not
	if err := kv.f.Sync(); err != nil {
		return 0, err
	}

	// advance offset for next append
	kv.writeOffset += int64(HeaderSize + len(key) + len(value))
	return offset, nil
}

func (kv *KVStore) readValueAt(offset int64) (string, error) {
	// read the fixed header first
	header := make([]byte, HeaderSize)
	if _, err := kv.f.ReadAt(header, offset); err != nil {
		return "", err
	}

	// op | record_len | key_len | value_len | key | value
	// 0 | 1..4 | 5..8 | 9..12

	// offset = 4
	// 4 | 5..8 | 9..12 | 13..16

	// op = 1 bytes
	// record_len = 4 bytes
	// key_len = 4 bytes
	// value_len = 4 bytes

	// read op type
	op := header[0]

	// this means it is a delete record
	if op == opDelete {
		return "", errors.New("Key does not exists")
	}

	// read key length
	key_len := binary.BigEndian.Uint32(header[5:9])
	// read value length
	value_len := binary.BigEndian.Uint32(header[9:13])

	value_offset := offset + int64(HeaderSize) + int64(key_len)
	value := make([]byte, value_len)

	if _, err := kv.f.ReadAt(value, value_offset); err != nil {
		return "", err
	}

	fmt.Println("I read this value = ", string(value))
	return string(value), nil
}

func (kv *KVStore) Set(key, value string) error {
	offset, err := kv.append(opPut, key, value)
	if err != nil {
		return err
	}

	kv.hashMap[key] = offset
	return nil
}

func (kv *KVStore) Get(key string) (string, error) {
	offset, ok := kv.hashMap[key]

	if !ok {
		return "", errors.New("Error key not found")
	}

	fmt.Println("Reading the key at offset: ", key, offset)
	return kv.readValueAt(offset)
}

func (kv *KVStore) Delete(key string) error {
	_, ok := kv.hashMap[key]
	if !ok {
		return errors.New("Error key not found")
	}

	// we just append so when log is replayed it will discard this key
	_, err := kv.append(opDelete, key, "")
	if err != nil {
		return err
	}

	// delete the key from hashmap now
	delete(kv.hashMap, key)
	return nil
}

func write() {
	f, err := os.Create("out.txt")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()
	f.WriteString("Hello Pranjal")
	f.Write([]byte("\nline2\nyess"))
	fmt.Println("wrote everything")
}

func writeInBinary(s string) {
	f, err := os.Create("out.bin")
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()
	// var value uint32 = 100000
	b := []byte(s)
	if err := binary.Write(f, binary.BigEndian, uint32(len(b))); err != nil {
		log.Fatal(err)
	}

	_, err2 := f.Write(b)
	if err2 != nil {
		log.Fatal(err2)
	}
}

func main() {
	kv := make(map[string]string)
	kv["hello"] = "world"

	fmt.Println("Hello, KV store")
	print(kv)
	write()
	writeInBinary("Hello\nPranjal")
	fmt.Println("***** Starting Operations *****")

	// 1. open the file for BOTH reading and writing, create if missing, append mode
	f, err := os.OpenFile(DefaultPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	// Get file info
	info, _ := f.Stat()

	// 2. build the struct
	kvStore := &KVStore{
		hashMap:     make(map[string]int64),
		f:           f,
		w:           bufio.NewWriter(f),
		writeOffset: info.Size(),
	}

	// write a couple of records
	if err := kvStore.Set("go", "1.0"); err != nil {
		log.Fatal(err)
	}
	if err := kvStore.Set("lang", "golang"); err != nil {
		log.Fatal(err)
	}
	if err := kvStore.Set("name", "Pranjal"); err != nil {
		log.Fatal(err)
	}

	// read them back
	v, err := kvStore.Get("go")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("go =", v)

	v, err = kvStore.Get("lang")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("lang =", v)

	v, err = kvStore.Get("name")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("name =", v)

	err = kvStore.Delete("name")

	v, err = kvStore.Get("name")
	if err != nil {
		fmt.Println(err)
	}

}
