package datastore

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDb_Int64(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	pairs := map[string]int64{
		"key1": 123,
		"key2": 345,
		"key3": 0,
	}

	outFile, err := os.Open(filepath.Join(dir, outFileName))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("putInt64/getInt64", func(t *testing.T) {
		for key, intValue := range pairs {
			err := db.PutInt64(key, intValue)
			if err != nil {
				t.Errorf("Cannot put %s: %s", key, err)
			}
			value, err := db.GetInt64(key)
			if err != nil {
				t.Errorf("Cannot get %s: %s", key, err)
			}
			if value != intValue {
				t.Errorf("Bad value returned expected %d, got %d", intValue, value)
			}
		}
	})

	outInfo, err := outFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	size1 := outInfo.Size()

	t.Run("file growth", func(t *testing.T) {
		for key, intValue := range pairs {
			err := db.PutInt64(key, intValue)
			if err != nil {
				t.Errorf("Cannot put %s: %s", key, err)
			}
		}
		outInfo, err := outFile.Stat()
		if err != nil {
			t.Fatal(err)
		}
		if size1*2 != outInfo.Size() {
			t.Errorf("Unexpected size (%d vs %d)", size1, outInfo.Size())
		}
	})

	t.Run("new db process", func(t *testing.T) {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		db, err = NewDb(dir)
		if err != nil {
			t.Fatal(err)
		}

		for key, intValue := range pairs {
			value, err := db.GetInt64(key)
			if err != nil {
				t.Errorf("Cannot put %s: %s", key, err)
			}
			if value != intValue {
				t.Errorf("Bad value returned expected %d, got %d", intValue, value)
			}
		}
	})
}

func TestDb_Segmentation(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	bufSize = 64 // 2 value in segment
	db, err := NewDb(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = db.Put("key1", "value11") //segment-1
	if err != nil {
		t.Errorf("error in writing key1 value11 : %s", err)
	}
	err = db.Put("key2", "value21") //segment-1
	if err != nil {
		t.Errorf("error in writing key2 value21 : %s", err)
	}

	err = db.Put("key3", "value31") //segment-2
	if err != nil {
		t.Errorf("error in writing key3 value31 : %s", err)
	}
	err = db.Put("key1", "value12") //segment-1 after segment-2 (no segment-2)
	if err != nil {
		t.Errorf("error in writing key1 value12 : %s", err)
	}

	err = db.Put("key3", "value32") //segment-2 after segment-3 (no segment-3)
	if err != nil {
		t.Errorf("error in writing key3 value32 : %s", err)
	}
	err = db.Put("key2", "value22") //segment-1 after segment-3 (no segment-3)
	if err != nil {
		t.Errorf("error in writing key2 value22 : %s", err)
	}
	//segment-3 dead

	err = db.Put("key4", "value41") //current-data
	if err != nil {
		t.Errorf("error in writing key4 value41 : %s", err)
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Errorf("not the appropriate number of files, there are %d should be %d", len(files), 3)
	}

	val, err := db.Get("key1")
	if err != nil {
		t.Errorf("error in read : %s", err)
	}
	if val != "value12" {
		t.Errorf("GET value is incorrect")
	}
	val, err = db.Get("key3")
	if err != nil {
		t.Errorf("error in read : %s", err)
	}
	if val != "value32" {
		t.Errorf("GET value is incorrect")
	}

}

func TestDb_Put(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	pairs := [][]string{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	outFile, err := os.Open(filepath.Join(dir, outFileName))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("put/get", func(t *testing.T) {
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
			value, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Cannot get %s: %s", pairs[0], err)
			}
			if value != pair[1] {
				t.Errorf("Bad value returned expected %s, got %s", pair[1], value)
			}
		}
	})

	outInfo, err := outFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	size1 := outInfo.Size()

	t.Run("file growth", func(t *testing.T) {
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
		}
		outInfo, err := outFile.Stat()
		if err != nil {
			t.Fatal(err)
		}
		if size1*2 != outInfo.Size() {
			t.Errorf("Unexpected size (%d vs %d)", size1, outInfo.Size())
		}
	})

	t.Run("new db process", func(t *testing.T) {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		db, err = NewDb(dir)
		if err != nil {
			t.Fatal(err)
		}

		for _, pair := range pairs {
			value, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
			if value != pair[1] {
				t.Errorf("Bad value returned expected %s, got %s", pair[1], value)
			}
		}
	})

}
