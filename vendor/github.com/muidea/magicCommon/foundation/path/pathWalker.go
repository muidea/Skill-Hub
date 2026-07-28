package path

import (
	"os"
	"path/filepath"
)

// WalkPath 遍历指定目录
func WalkPath(filePath string) ([]string, error) {
	fileList := []string{}
	err := filepath.Walk(filePath, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if f.IsDir() {
			return nil
		}

		fileList = append(fileList, path)
		return nil
	})

	return fileList, err
}

// ListPath 列出指定目录下的文件
// filePath 路径
// filterPattern 过滤规则
// recursive 是否递归
func ListPath(filePath string, filterPattern string, recursive bool) ([]string, error) {
	if recursive {
		return walkPathWithFilter(filePath, filterPattern)
	}
	return readDirWithFilter(filePath, filterPattern)
}

func walkPathWithFilter(root string, pattern string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}
		if matched {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func readDirWithFilter(dir string, pattern string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return nil, err
		}
		if matched {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	return files, nil
}
