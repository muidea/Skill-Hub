package path

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"

	fu "github.com/muidea/magicCommon/foundation/util"
)

type Op uint32

const (
	Create Op = 1 << iota
	Modify
	Remove
)

func (s Op) String() string {
	switch s {
	case Create:
		return "Create"
	case Modify:
		return "Modify"
	case Remove:
		return "Remove"
	}
	return "Unknown"
}

type Event struct {
	Path string
	Op   Op
}

type Observer interface {
	OnEvent(event Event)
}

// 监控指定目录及其子目录下的文件变化
// 包括新增、删除和修改
type Monitor struct {
	ignores      fu.StringSet
	syncMutex    sync.Mutex
	watchedPaths fu.StringSet
	fsWatcher    *fsnotify.Watcher
	observer     []Observer
	eventQueue   chan fsnotify.Event
	started      bool
	stopped      bool
	workerWg     sync.WaitGroup
}

func NewMonitor(ignores fu.StringSet) (*Monitor, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Monitor{
		ignores:      ignores,
		watchedPaths: fu.StringSet{},
		fsWatcher:    watcher,
	}, nil
}

func (s *Monitor) Start() error {
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()

	if s.started {
		return nil
	}
	if s.stopped {
		return errors.New("monitor already stopped")
	}

	s.eventQueue = make(chan fsnotify.Event, 1000)
	s.started = true

	s.workerWg.Add(1)
	go func() {
		defer s.workerWg.Done()
		for event := range s.eventQueue {
			s.handleEvent(event)
		}
	}()

	s.workerWg.Add(1)
	go func() {
		defer s.workerWg.Done()
		defer close(s.eventQueue)
		for {
			select {
			case event, ok := <-s.fsWatcher.Events:
				if !ok {
					return
				}
				select {
				case s.eventQueue <- event:
				default:
					//log.Warnf("event queue full, dropping event：%v", event)
				}
			case err, ok := <-s.fsWatcher.Errors:
				if !ok {
					return
				}
				_ = err
			}
		}
	}()

	return nil
}

func (s *Monitor) Stop() error {
	s.syncMutex.Lock()
	if s.stopped {
		s.syncMutex.Unlock()
		return nil
	}

	s.observer = []Observer{}
	s.watchedPaths = fu.StringSet{}
	s.started = false
	s.stopped = true
	s.syncMutex.Unlock()

	err := s.fsWatcher.Close()
	s.workerWg.Wait()
	return err
}

func (s *Monitor) AddObserver(observer Observer) {
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()

	s.observer = append(s.observer, observer)
}

func (s *Monitor) RemoveObserver(observer Observer) {
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()

	for i, o := range s.observer {
		if o == observer {
			s.observer = append(s.observer[:i], s.observer[i+1:]...)
			break
		}
	}
}

func (s *Monitor) AddIgnore(ignores fu.StringSet) {
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()

	for _, ignore := range ignores {
		s.ignores = s.ignores.Add(ignore)
	}
}

func (s *Monitor) isIgnore(path string) bool {
	for _, v := range s.ignores {
		if strings.Contains(path, v) {
			return true
		}
	}
	return false
}

func (s *Monitor) AddPath(path string) error {
	waitChannel := make(chan bool)
	defer close(waitChannel)
	go func() {
		s.refresh(path)
		waitChannel <- true
	}()
	<-waitChannel
	return nil
}

func (s *Monitor) RemovePath(path string) error {
	return s.removePath(path)
}

func (s *Monitor) handleEvent(event fsnotify.Event) {
	if s.isIgnore(event.Name) {
		return
	}

	//log.Infof("op:%s, path:%s", event.Op, event.Name)

	if s.isDir(event.Name) {
		s.pathEvent(event)
		return
	}

	s.fileEvent(event)
}

func (s *Monitor) isDir(path string) bool {
	// 如果文件存在，则直接判断文件是否是目录，否则判断文件是否在目录缓存中
	pathInfo, pathErr := os.Stat(path)
	if pathErr == nil {
		return pathInfo.IsDir()
	}

	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()
	return s.watchedPaths.Exist(path)
}

func (s *Monitor) pathEvent(event fsnotify.Event) {
	switch event.Op {
	case fsnotify.Create:
		if err := s.addPath(event.Name); err != nil {
			slog.Error("Failed to add path", "path", event.Name, "error", err)
		}
	case fsnotify.Write:
		return
	case fsnotify.Remove, fsnotify.Rename:
		_ = s.removePath(event.Name)
		return
	default:
		return
	}
	_ = filepath.WalkDir(event.Name, func(path string, info os.DirEntry, err error) error {
		if err != nil || s.isIgnore(path) {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		for _, observer := range s.snapshotObservers() {
			observer.OnEvent(Event{
				Path: path,
				Op:   Create,
			})
		}
		return nil
	})
}

func (s *Monitor) fileEvent(event fsnotify.Event) {
	var localEvent Event
	switch event.Op {
	case fsnotify.Create:
		localEvent = Event{
			Path: event.Name,
			Op:   Create,
		}
		_ = s.addPath(filepath.Dir(event.Name))
	case fsnotify.Write:
		localEvent = Event{
			Path: event.Name,
			Op:   Modify,
		}
	case fsnotify.Remove, fsnotify.Rename:
		localEvent = Event{
			Path: event.Name,
			Op:   Remove,
		}
	default:
		return
	}

	for _, observer := range s.snapshotObservers() {
		observer.OnEvent(localEvent)
	}
}

func (s *Monitor) addPath(path string) error {
	addFunc := func(addPath string) {
		if s.isIgnore(addPath) {
			return
		}
		s.syncMutex.Lock()
		defer s.syncMutex.Unlock()

		if s.watchedPaths.Exist(addPath) {
			return
		}

		//log.Warnf("add path:%s", addPath)
		if err := s.fsWatcher.Add(addPath); err != nil {
			return
		}
		s.watchedPaths = s.watchedPaths.Add(addPath)
	}

	addFunc(path)
	_ = filepath.WalkDir(path, func(subPath string, info os.DirEntry, err error) error {
		if err != nil || s.isIgnore(subPath) {
			return nil
		}
		if info.IsDir() {
			addFunc(subPath)
		}

		addFunc(filepath.Dir(subPath))
		return nil
	})

	return nil
}

func (s *Monitor) removePath(path string) error {
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()

	var toDelete []string
	for _, subPath := range s.watchedPaths {
		if IsSubPath(path, subPath) {
			toDelete = append(toDelete, subPath)
		}
	}

	for _, subPath := range toDelete {
		//log.Warnf("remove path: %s", subPath)
		s.watchedPaths = s.watchedPaths.Remove(subPath)
		if err := s.fsWatcher.Remove(subPath); err != nil {
			return err
		}
	}

	return nil
}

func (s *Monitor) refresh(path string) {
	_ = s.addPath(path)
	_ = filepath.WalkDir(path, func(path string, info os.DirEntry, err error) error {
		if err != nil || s.isIgnore(path) {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		for _, observer := range s.snapshotObservers() {
			observer.OnEvent(Event{
				Path: path,
				Op:   Create,
			})
		}
		return nil
	})
}

func (s *Monitor) snapshotObservers() []Observer {
	s.syncMutex.Lock()
	defer s.syncMutex.Unlock()

	observers := make([]Observer, len(s.observer))
	copy(observers, s.observer)
	return observers
}
