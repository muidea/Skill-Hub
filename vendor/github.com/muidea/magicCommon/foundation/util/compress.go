package util

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"log/slog"
)

func UnZipFile(zipFile, destDir string) (ret []string, err error) {
	zipHandler, zipErr := zip.OpenReader(zipFile)
	if zipErr != nil {
		err = zipErr
		slog.Error("UnZipFile failed", "error", zipErr.Error())
		return
	}
	defer func() {
		if closeErr := zipHandler.Close(); closeErr != nil {
			slog.Warn("Failed to close zip handler", "error", closeErr)
		}
	}()

	destRoot, absErr := filepath.Abs(destDir)
	if absErr != nil {
		return nil, absErr
	}

	for _, f := range zipHandler.File {
		path, pathErr := safeZipPath(destRoot, f.Name)
		if pathErr != nil {
			return nil, pathErr
		}

		if err = extractZipEntry(f, path); err != nil {
			return nil, err
		}

		ret = append(ret, path)
	}

	return
}

func ZipDir(sourceDir, outputFile string) error {
	newZipFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer func() { _ = newZipFile.Close() }()

	zipWriter := zip.NewWriter(newZipFile)
	defer func() { _ = zipWriter.Close() }()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对路径
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = relPath
		header.Method = zip.Deflate

		if info.IsDir() {
			header.Name += "/"
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if err = copyFileToZip(writer, path); err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func ZipFiles(files []string, outputFile string) error {
	newZipFile, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer func() { _ = newZipFile.Close() }()

	zipWriter := zip.NewWriter(newZipFile)
	defer func() { _ = zipWriter.Close() }()

	for _, file := range files {
		err = addFileToZip(zipWriter, file)
		if err != nil {
			return err
		}
	}

	return nil
}

func addFileToZip(zw *zip.Writer, filename string) error {
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() { _ = fileToZip.Close() }()

	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = filepath.Base(filename)
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, fileToZip)
	if err != nil {
		return err
	}

	return nil
}

func ZipFile(file, outputFile string) (err error) {
	return ZipFiles([]string{file}, outputFile)
}

func safeZipPath(destRoot, entryName string) (string, error) {
	targetPath := filepath.Join(destRoot, entryName)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}

	prefix := destRoot + string(os.PathSeparator)
	if absTarget != destRoot && !strings.HasPrefix(absTarget, prefix) {
		return "", fmt.Errorf("illegal zip entry path: %s", entryName)
	}

	return absTarget, nil
}

func extractZipEntry(f *zip.File, path string) error {
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(path, 0755); err != nil {
			slog.Error("UnZipFile failed", "path", path, "error", err.Error())
			return err
		}
		return nil
	}

	zfHandle, err := f.Open()
	if err != nil {
		slog.Error("getFieldReferenceValue failed", "file", f.Name, "error", err.Error())
		return err
	}
	defer func() {
		if closeErr := zfHandle.Close(); closeErr != nil {
			slog.Warn("Failed to close zip file handle", "file", f.Name, "error", closeErr)
		}
	}()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		slog.Error("getFieldReferenceValue failed", "path", filepath.Dir(path), "error", err.Error())
		return err
	}

	fHandle, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := fHandle.Close(); closeErr != nil {
			slog.Warn("Failed to close file handle", "path", path, "error", closeErr)
		}
	}()

	if _, err := io.Copy(fHandle, zfHandle); err != nil {
		slog.Error("UnZipFile failed", "error", err.Error())
		return err
	}

	return nil
}

func copyFileToZip(writer io.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(writer, file)
	return err
}
