package path

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"log/slog"
)

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsSubPath 判断subPath是否是topPath的子路径
func IsSubPath(topPath, subPath string) bool {
	// 转为绝对路径并清理
	absWorkspace, err := filepath.Abs(filepath.Clean(topPath))
	if err != nil {
		return false
	}
	absFile, err := filepath.Abs(filepath.Clean(subPath))
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(absWorkspace, absFile)
	if err != nil {
		return false
	}

	// 判断 rel 是否是以 `..` 开头，表示 filePath 不在 workspacePath 之内
	isSub := !strings.HasPrefix(rel, "..") && rel != ".."
	return isSub
}

// Exist 路径是否存在
func Exist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// IsDirEmpty 判断目录是否为空
func IsDirEmpty(dirPath string) (bool, error) {
	// 打开目录
	dir, err := os.Open(dirPath)
	if err != nil {
		return false, err
	}
	defer func() { _ = dir.Close() }()

	// 读取目录中的文件和子目录
	nameSlice, nameErr := dir.Readdirnames(1)
	if nameErr != nil && !errors.Is(nameErr, io.EOF) {
		return false, nameErr
	}

	return len(nameSlice) == 0, nil
}

func SplitParentDir(dirPath string) string {
	parentPath, _ := path.Split(dirPath)
	return strings.TrimRight(parentPath, "/")
}

func CleanPathContent(dirPath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		slog.Error("clean path content failed", "dirPath", dirPath, "error", err.Error())
		return
	}

	for _, entry := range entries {
		entryPath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			if err := os.RemoveAll(entryPath); err != nil {
				slog.Error("clean path content failed", "dirPath", entryPath, "error", err.Error())
			}
		} else {
			if err := os.Remove(entryPath); err != nil {
				slog.Error("clean path content failed", "file", entryPath, "error", err.Error())
			}
		}
	}
}

// CopyPath 深度复制目录结构及内容
// 特性：
// - 保留文件权限和修改时间
// - 并行文件复制（最大4并发）
// - 自动创建目标目录
// - 完善的错误处理链
func CopyPath(srcPath, dstPath string) error {
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return fmt.Errorf("检查源路径失败: %w", err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("%s 不是目录", srcPath)
	}

	if err := os.MkdirAll(dstPath, srcInfo.Mode()); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	entries, err := os.ReadDir(srcPath)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, 1)
	defer close(errChan)
	workerSem := make(chan struct{}, 4)
	defer close(workerSem)

	for _, entry := range entries {
		select {
		case err := <-errChan:
			return err
		default:
			workerSem <- struct{}{}
			wg.Add(1)

			go func(e fs.DirEntry) {
				defer func() {
					<-workerSem
					wg.Done()
				}()

				srcItem := filepath.Join(srcPath, e.Name())
				dstItem := filepath.Join(dstPath, e.Name())

				if e.IsDir() {
					if err := CopyPath(srcItem, dstItem); err != nil {
						trySendError(errChan, fmt.Errorf("子目录复制失败 %s: %w", srcItem, err))
					}
					return
				}

				if e.Type()&fs.ModeSymlink != 0 {
					if err := handleSymlink(srcItem, dstItem); err != nil {
						trySendError(errChan, err)
					}
					return
				}

				if err := copyFile(srcItem, dstItem); err != nil {
					trySendError(errChan, fmt.Errorf("文件复制失败 %s: %w", srcItem, err))
				}
			}(entry)
		}
	}

	wg.Wait()

	if err := <-errChan; err != nil {
		return err
	}
	return nil
}

// copyFile 带元数据保留的文件复制
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.CopyBuffer(dstFile, srcFile, make([]byte, 32*1024)); err != nil {
		return fmt.Errorf("内容复制失败: %w", err)
	}

	if err := os.Chtimes(dst, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		return fmt.Errorf("保留修改时间失败: %w", err)
	}
	return nil
}

// handleSymlink 处理符号链接（不跟随）
func handleSymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return fmt.Errorf("读取链接失败: %w", err)
	}
	return os.Symlink(target, dst)
}

// trySendError 非阻塞错误传递
func trySendError(ch chan<- error, err error) {
	select {
	case ch <- err:
	default:
	}
}
