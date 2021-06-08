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
	"sync"
)

const outFileName = "current-data"
const segmentName = "segment-"

var bufSize = 10 * 1024 * 1024

const metaDataSize = 16

var ErrNotFound = fmt.Errorf("record does not exist")
var WrongDataType = fmt.Errorf("wrong data type")

type hashIndex map[string]int64

//var segments = make(map[string]*Db)

var directory = "."

type Db struct {
	out       *os.File // opened file
	outPath   string   //path to file where we reade inf
	outOffset int64    //size
	dbLock    *sync.RWMutex
	segments  map[string]*Db
	index     hashIndex //map of hash to value
}

func NewDb(dir string) (*Db, error) {
	directory = dir
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if strings.Contains(file.Name(), segmentName) || outFileName == file.Name() {
			db, err := fillMap(file.Name())
			if err != nil {
				return nil, err
			}
			db.segments[file.Name()] = db //!!!!!
		}
	}
	db, err := fillMap(outFileName)
	if err != nil {
		return nil, err
	}
	db.segments[outFileName] = db
	return db.segments[outFileName], nil
}

func fillMap(name string) (*Db, error) {
	outputPath := filepath.Join(directory, name)
	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	db := &Db{
		outPath:  outputPath,
		out:      f,
		index:    make(hashIndex),
		dbLock:   new(sync.RWMutex),
		segments: make(map[string]*Db),
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

	buf := make([]byte, 0, bufSize)
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

		if size < uint32(bufSize) {
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

func (db *Db) getFromOne(key string) (string, string, error) {
	position, ok := db.index[key]
	if !ok {
		return "", "", ErrNotFound
	}

	file, err := os.Open(db.outPath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	_, err = file.Seek(position, 0)
	if err != nil {
		return "", "", err
	}

	reader := bufio.NewReader(file)
	value, typeValue, err := readValue(reader)
	if err != nil {
		return "", "", err
	}
	return value, typeValue, nil
}

func (db *Db) GetInt64(key string) (int64, error) {
	db.dbLock.RLock()
	defer db.dbLock.RUnlock()
	counter := 0
	for _, currentDB := range db.segments {
		counter++
		value, typeValue, finalError := currentDB.getFromOne(key)
		if finalError == nil && typeValue == "i" {
			res, err := strconv.Atoi(value)
			if err != nil {
				return -1, WrongDataType
			}
			return int64(res), nil
		}
		if counter == len(db.segments) {
			return -1, finalError
		}
	}
	return -1, ErrNotFound
}

func (db *Db) Get(key string) (string, error) {
	db.dbLock.RLock()
	defer db.dbLock.RUnlock()
	counter := 0
	for _, currentDB := range db.segments {
		counter++
		value, typeValue, finalError := currentDB.getFromOne(key)
		if finalError == nil && typeValue == "s" {
			return value, nil
		}
		if counter == len(db.segments) {
			return "", finalError
		}
	}
	return "", ErrNotFound
}

func (db *Db) putFromOne(key, value, typeValue string) error {
	if typeValue != "s" && typeValue != "i" {
		return WrongDataType
	}
	e := entry{
		key:       key,
		value:     value,
		typeValue: typeValue,
	}
	n, err := db.out.Write(e.Encode())
	if err == nil {
		db.index[key] = db.outOffset
		db.outOffset += int64(n)
	}
	return err
}

func (db *Db) PutInt64(key string, value int64) error {
	db.dbLock.RLock()
	defer db.dbLock.RUnlock()
	s := fmt.Sprintf("%d", value)
	if int(db.outOffset)+len(key)+len(s)+metaDataSize >= bufSize {
		err := db.segmentation()
		if err != nil {
			return err
		}
	}
	return db.putFromOne(key, s, "i")
}

func (db *Db) Put(key, value string) error {
	db.dbLock.RLock()
	defer db.dbLock.RUnlock()
	if int(db.outOffset)+len(key)+len(value)+metaDataSize >= bufSize {
		err := db.segmentation()
		if err != nil {
			return err
		}
	}
	return db.putFromOne(key, value, "s")
}

func (db *Db) segmentation() error {
	type normKV = struct {
		key   string
		value string
		typeV string
	}
	type normVT = struct {
		value string
		typeV string
	}
	isChangedSegment := make(map[string][]normKV)
	noDeletedKeys := make(map[string]bool)
	for k := range db.index {
		isFind := false
		for sk, sv := range db.segments {
			_, find := sv.index[k]
			if sk != outFileName && find {
				isFind = true
				value, typeValue, err := db.getFromOne(k)
				if err != nil {
					return err
				}
				isChangedSegment[sk] = append(isChangedSegment[sk], normKV{key: k, value: value, typeV: typeValue})
				break
			}
		}
		if !isFind {
			noDeletedKeys[k] = true
		}
	}

	for sName, norms := range isChangedSegment {
		normSegmentValues := make(map[string]normVT)
		for k := range db.segments[sName].index {
			val, typeValue, err := db.segments[sName].getFromOne(k)
			if err != nil {
				return err
			}
			normSegmentValues[k] = normVT{value: val, typeV: typeValue}
		}

		for _, obj := range norms {
			normSegmentValues[obj.key] = normVT{value: obj.value, typeV: obj.typeV}
		}

		err := os.Truncate(filepath.Join(db.segments[sName].outPath), 0)
		db.segments[sName].outOffset = 0
		if err != nil {
			return err
		}
		for k, v := range normSegmentValues {
			if v.typeV == "s" {
				err = db.segments[sName].putFromOne(k, v.value, "s")
				if err != nil {
					return err
				}
			} else if v.typeV == "i" {
				_, err := strconv.Atoi(v.value)
				if err != nil {
					return WrongDataType
				}
				err = db.segments[sName].putFromOne(k, v.value, "i")
			} else {
				return WrongDataType
			}
		}
	}

	if len(noDeletedKeys) != 0 {
		segment := segmentName + strconv.Itoa(len(db.segments))
		newDb, err := fillMap(segment)
		if err != nil {
			return err
		}
		for key := range noDeletedKeys {
			val, typeValue, err := db.getFromOne(key)
			if err != nil {
				return err
			}
			if typeValue == "s" {
				err = newDb.putFromOne(key, val, "s")
				if err != nil {
					return err
				}
			} else if typeValue == "i" {
				_, err := strconv.Atoi(val)
				if err != nil {
					return WrongDataType
				}
				err = newDb.putFromOne(key, val, "i")
			} else {
				return WrongDataType
			}
		}
		db.segments[segment] = newDb
	}

	err := os.Truncate(filepath.Join(db.outPath), 0)
	db.outOffset = 0
	if err != nil {
		return err
	}
	db.index = make(hashIndex)

	return nil
}
