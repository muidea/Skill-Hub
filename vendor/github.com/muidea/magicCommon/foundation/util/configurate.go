package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func LoadConfig(filePath string, ptr interface{}) (err error) {
	if ptr == nil {
		err = fmt.Errorf("illegal ptr")
		return
	}

	filePtr, fileErr := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if fileErr != nil {
		err = fileErr
		return
	}
	defer func() { _ = filePtr.Close() }()

	byteContent, byteErr := io.ReadAll(filePtr)
	if byteErr != nil {
		err = byteErr
		return
	}

	err = json.Unmarshal(byteContent, ptr)
	return
}

func SaveConfig(filePath string, ptr interface{}) (err error) {
	if ptr == nil {
		return
	}

	byteContent, byteErr := json.Marshal(ptr)
	if byteErr != nil {
		err = byteErr
		return
	}

	fileHandle, fileErr := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if fileErr != nil {
		err = fileErr
		return
	}
	defer func() { _ = fileHandle.Close() }()

	var byteBuffer bytes.Buffer
	err = json.Indent(&byteBuffer, byteContent, "", "\t")
	if err != nil {
		return
	}

	_, err = byteBuffer.WriteTo(fileHandle)
	return
}
