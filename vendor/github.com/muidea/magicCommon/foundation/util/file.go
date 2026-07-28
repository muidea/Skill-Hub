package util

import (
	"io"
	"os"

	"log/slog"
)

func CopyFile(srcFile, dstFile string, delSource bool) (err error) {
	srcFileHandle, err := os.Open(srcFile)
	if err != nil {
		slog.Error("CopyFile failed", "rawFile", srcFile, "error", err.Error())
		return
	}
	defer func() { _ = srcFileHandle.Close() }()

	dstFileHandle, err := os.Create(dstFile)
	if err != nil {
		slog.Error("CopyFile failed", "filePath", dstFile, "error", err.Error())
		return
	}
	defer func() { _ = dstFileHandle.Close() }()

	_, err = io.Copy(dstFileHandle, srcFileHandle)
	if err != nil {
		slog.Error("CopyFile failed", "filePath", dstFile, "error", err.Error())
		return
	}

	err = dstFileHandle.Sync()
	if err != nil {
		slog.Error("CopyFile failed", "filePath", dstFile, "error", err.Error())
		return
	}

	if delSource {
		err = os.Remove(srcFile)
		if err != nil {
			slog.Error("CopyFile failed", "rawFile", srcFile, "error", err.Error())
			return
		}
	}

	return
}
