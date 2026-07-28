package net

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"log/slog"
)

// MultipartFormFile 从 HTTP 请求中提取指定的文件，并将其保存到指定的路径。
// req 是 HTTP 请求。
// fieldName 是表单中文件字段的名称。
// dstFilePath 是文件将被保存的目录路径。
// fileName 是指定保存的文件名，如果值为空，则使用原始文件名。
// 返回值 ret 是上传文件的名称，err 是错误信息（如果有）。
func MultipartFormFile(req *http.Request, fieldName, dstFilePath, fileName string) (ret string, err error) {
	// 从请求中获取文件内容和文件头信息。
	fileContent, fileHead, fileErr := req.FormFile(fieldName)
	if fileErr != nil {
		err = fileErr
		slog.Error("get file field failed, field: fieldName, err: err.Error(", "field", fieldName, "error", err.Error())
		return
	}
	defer func() { _ = fileContent.Close() }()

	if fileName == "" {
		fileName = fileHead.Filename
	}

	// 验证 dstFilePath 是否为合法的目录路径
	if !isValidDirectory(dstFilePath) {
		err = fmt.Errorf("invalid destination directory: %s", dstFilePath)
		slog.Error("invalid destination directory, err: err.Error(", "field", err.Error())
		return
	}

	// 验证文件名是否合法
	if !isValidFileName(fileName) {
		err = fmt.Errorf("invalid file name: %s", fileName)
		slog.Error("invalid file name, err: err.Error(", "field", err.Error())
		return
	}

	// 构建目标文件的完整路径
	dstFullFilePath := filepath.Join(dstFilePath, fileName)
	// 创建目标文件
	dstFileHandle, dstFileErr := os.Create(dstFullFilePath)
	if dstFileErr != nil {
		err = dstFileErr
		slog.Error("create destination file failed, err: err.Error(", "field", err.Error())
		return
	}
	defer func() { _ = dstFileHandle.Close() }()

	// 将文件内容从请求复制到目标文件中
	_, err = io.Copy(dstFileHandle, fileContent)
	if err != nil {
		slog.Error("copy destination file failed, err: err.Error(", "field", err.Error())
		return
	}

	// 设置返回值为文件名
	ret = fileName
	return
}

// isValidDirectory 验证路径是否为合法的目录
func isValidDirectory(path string) bool {
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return false
	}

	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(cleanPath, 0755)
			return err == nil
		}
		return false
	}
	return info.IsDir()
}

// isValidFileName 验证文件名是否合法
func isValidFileName(name string) bool {
	return len(name) > 0 && !strings.ContainsAny(name, `\/:*?"<>|`)
}
