package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const outFileName = "current-data"
const segmentName = "segment-"
const bufSize = 64
const metaDataSize = 12

var ErrNotFound = fmt.Errorf("record does not exist")

type hashIndex map[string]int64

var segments = make(map[string]*Db)

var directory = "."

type Db struct {
	out       *os.File // opened file
	outPath   string   //path to file where we reade inf
	outOffset int64    //size

	index hashIndex //map of hash to value
}

func NewDb(dir string) (*Db, error) {
	directory = dir
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	isFindOut := false
	for _, file := range files {
		if strings.Contains(file.Name(), segmentName) || outFileName == file.Name() {
			if outFileName == file.Name() {
				isFindOut = true
			}
			db, err := fillMap(file.Name())
			if err != nil {
				return nil, err
			}
			segments[file.Name()] = db
		}
	}
	if !isFindOut {
		db, err := fillMap(outFileName)
		if err != nil {
			return nil, err
		}
		segments[outFileName] = db
	}
	return segments[outFileName], nil
}

func fillMap(name string) (*Db, error) {
	outputPath := filepath.Join(directory, name)
	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	db := &Db{
		outPath: outputPath,
		out:     f,
		index:   make(hashIndex),
	}
	err = db.recover()
	if err != nil && err != io.EOF {
		return nil, err
	}
	return db, nil
}

func (db *Db) recover() error {
	input, err := os.Open(db.outPath)
	if err != nil {
		return err
	}
	defer input.Close()

	var buf [bufSize]byte
	in := bufio.NewReaderSize(input, bufSize)
	for err == nil {
		var (
			header, data []byte
			n            int
		)
		header, err = in.Peek(bufSize)
		if err == io.EOF {
			if len(header) == 0 {
				return err
			}
		} else if err != nil {
			return err
		}
		size := binary.LittleEndian.Uint32(header)

		if size < bufSize {
			data = buf[:size]
		} else {
			data = make([]byte, size)
		}
		n, err = in.Read(data)

		if err == nil {
			if n != int(size) {
				return fmt.Errorf("corrupted file")
			}

			var e entry
			e.Decode(data)
			db.index[e.key] = db.outOffset
			db.outOffset += int64(n)
		}
	}
	return err
}

func (db *Db) Close() error {
	return db.out.Close()
}

func (db *Db) getFromOne(key string) (string, error) {
	position, ok := db.index[key]
	if !ok {
		return "", ErrNotFound
	}

	file, err := os.Open(db.outPath)
	if err != nil {
		fmt.Println("error g2")
		return "", err
	}
	defer file.Close()

	_, err = file.Seek(position, 0)
	if err != nil {
		fmt.Println("error g3")
		return "", err
	}

	reader := bufio.NewReader(file)
	value, err := readValue(reader)
	if err != nil {
		fmt.Println("error g4")
		return "", err
	}
	return value, nil
}

func (db *Db) Get(key string) (string, error) {
	counter := 0
	for _, currentDB := range segments {
		//fmt.Println(counter)
		counter++
		value, finalError := currentDB.getFromOne(key)
		if finalError == nil {
			return value, nil
		}
		if counter == len(segments) {
			return "", finalError
		}
	}
	return "", ErrNotFound
}

func (db *Db) Put(key, value string) error {
	if int(db.outOffset)+len(key)+len(value)+metaDataSize >= bufSize {
		err := db.segmentation()
		if err != nil {
			fmt.Println("error p1")
			return err
		}
	}
	e := entry{
		key:   key,
		value: value,
	}
	n, err := db.out.Write(e.Encode())
	if err == nil {
		db.index[key] = db.outOffset
		db.outOffset += int64(n)
	}
	return err
}

func (db *Db) segmentation() error {
	type normKV = struct {
		key   string
		value string
	}
	isChangedSegment := make(map[string][]normKV)
	noDeletedKeys := make(map[string]bool)
	for k := range db.index {
		for sk, sv := range segments {
			if sk != outFileName {
				_, find := sv.index[k]
				if find {
					value, err := db.getFromOne(k)
					if err != nil {
						fmt.Println("error 2")
						return err
					}
					isChangedSegment[sk] = append(isChangedSegment[sk], normKV{key: k, value: value})
					break
				}
			}
			noDeletedKeys[k] = true
		}
	}

	for sName, norms := range isChangedSegment {
		fmt.Println(norms)
		normSegmentValues := make(map[string]string)
		for k := range segments[sName].index {
			val, err := segments[sName].getFromOne(k)
			if err != nil {
				return err
			}
			normSegmentValues[k] = val
		}

		for _, obj := range norms {
			normSegmentValues[obj.key] = obj.value
		}

		err := os.Truncate(filepath.Join(segments[sName].outPath), 0)
		segments[sName].outOffset = 0
		if err != nil {
			return err
		}
		for k,v := range normSegmentValues {
			err := segments[sName].Put(k,v)
			if err != nil {
				fmt.Println("error here")
				return err
			}
		}
	}

	if len(noDeletedKeys) != 0 {
		segment := segmentName + strconv.Itoa(len(segments))
		newDb, err := fillMap(segment)
		if err != nil {
			fmt.Println("error 5")
			return err
		}
		for key := range noDeletedKeys {
			val, err := db.Get(key)
			if err != nil {
				fmt.Println("error 61")
				return err
			}
			err = newDb.Put(key, val)
			if err != nil {
				fmt.Println("error 6")
				return err
			}
		}
		segments[segment] = newDb
	}

	err := os.Truncate(filepath.Join(db.outPath), 0)
	db.outOffset = 0
	if err != nil {
		fmt.Println("error 7")
		return err
	}
	db.index = make(hashIndex)

	return nil
}
