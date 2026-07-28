package configuration

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"log/slog"
)

// FileWatcherImpl 文件监听器实现
type FileWatcherImpl struct {
	watcher      *fsnotify.Watcher
	callbacks    map[string][]func()
	dirCallbacks map[string][]func(string)
	mu           sync.RWMutex
	closed       bool
}

// NewFileWatcher 创建文件监听器
func NewFileWatcher() (*FileWatcherImpl, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	fw := &FileWatcherImpl{
		watcher:      watcher,
		callbacks:    make(map[string][]func()),
		dirCallbacks: make(map[string][]func(string)),
		closed:       false,
	}

	// 启动事件处理循环
	go fw.eventLoop()

	return fw, nil
}

// Watch 监听文件变化
func (fw *FileWatcherImpl) Watch(filePath string, callback func()) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return fmt.Errorf("file watcher is closed")
	}

	// 添加文件到监听器
	if err := fw.watcher.Add(filePath); err != nil {
		return fmt.Errorf("failed to watch file: %w", err)
	}

	// 注册回调函数
	if _, exists := fw.callbacks[filePath]; !exists {
		fw.callbacks[filePath] = make([]func(), 0)
	}
	fw.callbacks[filePath] = append(fw.callbacks[filePath], callback)

	return nil
}

// WatchDirectory 监听目录变化
func (fw *FileWatcherImpl) WatchDirectory(dirPath string, callback func(filePath string)) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return fmt.Errorf("file watcher is closed")
	}

	// 添加目录到监听器
	if err := fw.watcher.Add(dirPath); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	// 注册回调函数
	if _, exists := fw.dirCallbacks[dirPath]; !exists {
		fw.dirCallbacks[dirPath] = make([]func(string), 0)
	}
	fw.dirCallbacks[dirPath] = append(fw.dirCallbacks[dirPath], callback)

	return nil
}

// Unwatch 取消监听文件
func (fw *FileWatcherImpl) Unwatch(filePath string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return fmt.Errorf("file watcher is closed")
	}

	// 从监听器中移除文件
	if err := fw.watcher.Remove(filePath); err != nil {
		return fmt.Errorf("failed to unwatch file: %w", err)
	}

	// 删除回调函数
	delete(fw.callbacks, filePath)

	return nil
}

// UnwatchDirectory 取消监听目录
func (fw *FileWatcherImpl) UnwatchDirectory(dirPath string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return fmt.Errorf("file watcher is closed")
	}

	// 从监听器中移除目录
	if err := fw.watcher.Remove(dirPath); err != nil {
		return fmt.Errorf("failed to unwatch directory: %w", err)
	}

	// 删除回调函数
	delete(fw.dirCallbacks, dirPath)

	return nil
}

// Close 关闭文件监听器
func (fw *FileWatcherImpl) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.closed {
		return nil
	}

	fw.closed = true
	return fw.watcher.Close()
}

// eventLoop 事件处理循环
func (fw *FileWatcherImpl) eventLoop() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			slog.Error("File watcher error", "error", err)
		}
	}
}

// handleEvent 处理文件系统事件
func (fw *FileWatcherImpl) handleEvent(event fsnotify.Event) {
	fw.mu.RLock()
	defer fw.mu.RUnlock()

	// 只处理写入和创建事件
	if event.Op&fsnotify.Write == 0 && event.Op&fsnotify.Create == 0 {
		return
	}

	filePath := event.Name

	// 检查是否是文件事件
	if callbacks, exists := fw.callbacks[filePath]; exists {
		for _, callback := range callbacks {
			go callback()
		}
	}

	// 检查是否是目录事件
	for dirPath, callbacks := range fw.dirCallbacks {
		if filepath.Dir(filePath) == dirPath {
			for _, callback := range callbacks {
				go callback(filePath)
			}
		}
	}
}

// SimpleFileWatcher 简单的文件监听器（基于轮询）
type SimpleFileWatcher struct {
	fileModTimes map[string]time.Time
	callbacks    map[string][]func()
	dirCallbacks map[string][]func(string)
	mu           sync.RWMutex
	stopChan     chan struct{}
	interval     time.Duration
	stopOnce     sync.Once
}

// NewSimpleFileWatcher 创建简单的文件监听器
func NewSimpleFileWatcher(interval time.Duration) *SimpleFileWatcher {
	return &SimpleFileWatcher{
		fileModTimes: make(map[string]time.Time),
		callbacks:    make(map[string][]func()),
		dirCallbacks: make(map[string][]func(string)),
		stopChan:     make(chan struct{}),
		interval:     interval,
	}
}

// Watch 监听文件变化
func (sw *SimpleFileWatcher) Watch(filePath string, callback func()) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// 获取文件的修改时间
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	sw.fileModTimes[filePath] = info.ModTime()

	// 注册回调函数
	if _, exists := sw.callbacks[filePath]; !exists {
		sw.callbacks[filePath] = make([]func(), 0)
	}
	sw.callbacks[filePath] = append(sw.callbacks[filePath], callback)

	return nil
}

// WatchDirectory 监听目录变化
func (sw *SimpleFileWatcher) WatchDirectory(dirPath string, callback func(filePath string)) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// 注册回调函数
	if _, exists := sw.dirCallbacks[dirPath]; !exists {
		sw.dirCallbacks[dirPath] = make([]func(string), 0)
	}
	sw.dirCallbacks[dirPath] = append(sw.dirCallbacks[dirPath], callback)

	return nil
}

// Start 启动监听
func (sw *SimpleFileWatcher) Start() {
	go sw.pollLoop()
}

// Stop 停止监听
func (sw *SimpleFileWatcher) Stop() {
	sw.stopOnce.Do(func() {
		close(sw.stopChan)
	})
}

// Close 关闭文件监听器
func (sw *SimpleFileWatcher) Close() error {
	sw.Stop()
	return nil
}

// pollLoop 轮询循环
func (sw *SimpleFileWatcher) pollLoop() {
	ticker := time.NewTicker(sw.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sw.checkFiles()
		case <-sw.stopChan:
			return
		}
	}
}

// checkFiles 检查文件变化
func (sw *SimpleFileWatcher) checkFiles() {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// 检查文件变化
	for filePath, oldModTime := range sw.fileModTimes {
		info, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		newModTime := info.ModTime()
		if newModTime.After(oldModTime) {
			sw.fileModTimes[filePath] = newModTime
			if callbacks, exists := sw.callbacks[filePath]; exists {
				for _, callback := range callbacks {
					go callback()
				}
			}
		}
	}

	// 检查目录变化（简化实现）
	// 在实际应用中，可能需要更复杂的目录变化检测
}
