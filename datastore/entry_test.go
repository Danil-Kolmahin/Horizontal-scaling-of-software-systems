package datastore

import (
	"bufio"
	"bytes"
	"testing"
)

func TestEntry_Encode(t *testing.T) {
	e := entry{"key", "value", "s"}
	e.Decode(e.Encode())
	if e.key != "key" {
		t.Error("incorrect key")
	}
	if e.value != "value" {
		t.Error("incorrect value")
	}
	if e.typeValue != "s" {
		t.Error("incorrect type")
	}
}

func TestReadValue(t *testing.T) {
	e := entry{"key", "test-value", "st"}
	data := e.Encode()
	v, ty, err := readValue(bufio.NewReader(bytes.NewReader(data)))
	if err != nil {
		t.Fatal(err)
	}
	if v != "test-value" {
		t.Errorf("Got bat value [%s]", v)
	}
	if ty != "st" {
		t.Errorf("Got bat type [%s]", ty)
	}
}
