package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
)

type entry struct {
	key, value string
	typeValue  string //
}

func (e *entry) Encode() []byte {
	kl := len(e.key)
	vl := len(e.value)
	tl := len(e.typeValue)    //
	size := kl + vl + 16 + tl //
	res := make([]byte, size)
	binary.LittleEndian.PutUint32(res, uint32(size))
	binary.LittleEndian.PutUint32(res[4:], uint32(kl))
	copy(res[8:], e.key)
	binary.LittleEndian.PutUint32(res[kl+8:], uint32(vl))
	copy(res[kl+12:], e.value)
	binary.LittleEndian.PutUint32(res[kl+vl+12:], uint32(tl))
	copy(res[kl+vl+16:], e.typeValue)
	return res
}

func (e *entry) Decode(input []byte) {
	kl := binary.LittleEndian.Uint32(input[4:])
	keyBuf := make([]byte, kl)
	copy(keyBuf, input[8:kl+8])
	e.key = string(keyBuf)

	vl := binary.LittleEndian.Uint32(input[kl+8:])
	valBuf := make([]byte, vl)
	copy(valBuf, input[kl+12:kl+12+vl])
	e.value = string(valBuf)

	tl := binary.LittleEndian.Uint32(input[kl+vl+12:])

	typeBuf := make([]byte, tl)
	copy(typeBuf, input[kl+vl+16:kl+16+vl+tl])
	e.typeValue = string(typeBuf)

}

func readValue(in *bufio.Reader) (string, string, error) {
	header, err := in.Peek(8)
	if err != nil {
		fmt.Println("error e1")
		return "", "", err
	}
	keySize := int(binary.LittleEndian.Uint32(header[4:]))
	_, err = in.Discard(keySize + 8)
	if err != nil {
		fmt.Println("error e2")
		return "", "", err
	}

	header, err = in.Peek(4)
	if err != nil {
		fmt.Println("error e3")
		return "", "", err
	}
	valSize := int(binary.LittleEndian.Uint32(header))
	_, err = in.Discard(4)
	if err != nil {
		fmt.Println("error e4")
		return "", "", err
	}

	dataValue := make([]byte, valSize)
	n, err := in.Read(dataValue)
	if err != nil {
		fmt.Println("error e5")
		return "", "", err
	}
	if n != valSize {
		fmt.Println("error e6")
		return "", "", fmt.Errorf("can't read value bytes (read %d, expected %d)", n, valSize)
	}

	header, err = in.Peek(4)
	if err != nil {
		fmt.Println("error e3")
		return "", "", err
	}
	typeSize := int(binary.LittleEndian.Uint32(header))
	_, err = in.Discard(4)
	if err != nil {
		fmt.Println("error e4")
		return "", "", err
	}

	dataType := make([]byte, typeSize)
	n, err = in.Read(dataType)
	if err != nil {
		fmt.Println("error e5")
		return "", "", err
	}
	if n != typeSize {
		fmt.Println("error e6")
		return "", "", fmt.Errorf("can't read value bytes (read %d, expected %d)", n, valSize)
	}

	return string(dataValue), string(dataType), nil
}
