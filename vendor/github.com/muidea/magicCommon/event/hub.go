package event

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"

	cd "github.com/muidea/magicCommon/def"
	"github.com/muidea/magicCommon/execute"
	"github.com/muidea/magicCommon/foundation/util"
)

type Values map[string]any

func (s Values) Set(key string, value any) {
	s[key] = value
}

func (s Values) Get(key string) any {
	val, ok := s[key]
	if ok {
		return val
	}

	return nil
}

func (s Values) GetString(key string) string {
	return GetTypedValue[string](s, key, "", "string")
}

func (s Values) GetInt(key string) int {
	return GetTypedValue[int](s, key, 0, "int")
}

func (s Values) GetBool(key string) bool {
	return GetTypedValue[bool](s, key, false, "bool")
}

// GetTypedValue 泛型方法获取指定类型的值
func GetTypedValue[T any](s Values, key string, defaultValue T, typeName string) T {
	val := s.Get(key)
	if val == nil {
		return defaultValue
	}

	if v, ok := val.(T); ok {
		return v
	}

	slog.Warn("illegal value, not expected type", "type", typeName, "value", val)
	return defaultValue
}

type Event interface {
	ID() string
	Source() string
	Destination() string
	LaneKey() string
	Header() Values
	Context() context.Context
	BindContext(ctx context.Context)
	BindLaneKey(laneKey string)
	Data() any
	SetData(key string, val any)
	GetData(key string) any
	Match(pattern string) bool
}

type Result interface {
	Error() *cd.Error
	Set(data any, err *cd.Error)
	Get() (any, *cd.Error)
	SetVal(key string, val any)
	GetVal(key string) any
}

type Observer interface {
	ID() string
	Notify(event Event, result Result)
}

type SimpleObserver interface {
	Observer
	Subscribe(eventID string, observerFunc ObserverFunc)
	Unsubscribe(eventID string)
}

type Hub interface {
	Subscribe(eventID string, observer Observer)
	Unsubscribe(eventID string, observer Observer)
	Post(event Event)
	Send(event Event) Result
	Terminate(ctx context.Context)
}

// HubOption Hub 配置项，用于控制内部缓冲和并发策略
type HubOption func(*hubOptions)

type hubOptions struct {
	perLaneChanSize   int
	hubActionChanSize int
	workerPoolSize    int
	laneIdleTimeout   time.Duration
}

const defaultMaxPerLaneChanSize = 64
const defaultMaxHubActionChanSize = 1024
const defaultMaxWorkerPoolSize = 256
const defaultLaneIdleTimeout = 30 * time.Second

func defaultHubOptions(capacitySize int) *hubOptions {
	if capacitySize <= 0 {
		capacitySize = 1
	}

	perLaneChanSize := capacitySize
	if perLaneChanSize > defaultMaxPerLaneChanSize {
		perLaneChanSize = defaultMaxPerLaneChanSize
	}

	return &hubOptions{
		// hubActionChannel 仅承担订阅控制面，不应跟随 500000 之类的大容量配置常驻扩张。
		// per-lane channel 若跟随 500000 之类的容量，会在每个新 lane 上常驻数 MB 内存。
		// workerPoolSize 控制同源事件异步投递的 goroutine 并发上限，需要与 hub 总队列容量解耦。
		perLaneChanSize:   perLaneChanSize,
		hubActionChanSize: minInt(capacitySize, defaultMaxHubActionChanSize),
		workerPoolSize:    minInt(capacitySize, defaultMaxWorkerPoolSize),
		laneIdleTimeout:   defaultLaneIdleTimeout,
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}

	return right
}

// WithPerDestinationChanSize 兼容旧命名，配置每个 lane 的内部 actionChannel 缓冲大小。
func WithPerDestinationChanSize(size int) HubOption {
	return WithPerLaneChanSize(size)
}

// WithPerLaneChanSize 配置每个顺序 lane 的内部 actionChannel 缓冲大小。
func WithPerLaneChanSize(size int) HubOption {
	return func(o *hubOptions) {
		if size > 0 {
			o.perLaneChanSize = size
		}
	}
}

// WithHubActionChanSize 配置 Hub 级别的 actionChannel 缓冲大小
func WithHubActionChanSize(size int) HubOption {
	return func(o *hubOptions) {
		if size > 0 {
			o.hubActionChanSize = size
		}
	}
}

// WithWorkerPoolSize 配置内部 Execute 的 worker 池大小
func WithWorkerPoolSize(size int) HubOption {
	return func(o *hubOptions) {
		if size > 0 {
			o.workerPoolSize = size
		}
	}
}

// WithLaneIdleTimeout 配置每个 lane 空闲多久后自动回收。
func WithLaneIdleTimeout(timeout time.Duration) HubOption {
	return func(o *hubOptions) {
		if timeout > 0 {
			o.laneIdleTimeout = timeout
		}
	}
}

type ObserverList []Observer
type ID2ObserverMap map[string]ObserverList
type ObserverFunc func(Event, Result)
type ID2ObserverFuncMap map[string]ObserverFunc
type actionChannel chan action
type LaneKey2ActionChannelMap map[string]*laneActionChannel

type laneEnqueueResult int

const (
	laneEnqueueOK laneEnqueueResult = iota
	laneEnqueueClosed
	laneEnqueueTimeout
)

type laneActionChannel struct {
	key        string
	ch         actionChannel
	mu         sync.Mutex
	closed     bool
	lastActive atomic.Int64
}

type laneExecutionContextKey struct{}

type laneContextEvent struct {
	Event
	ctx context.Context
}

func (s *laneContextEvent) Context() context.Context {
	if s.ctx == nil {
		return context.Background()
	}

	return s.ctx
}

func (s *laneContextEvent) BindContext(ctx context.Context) {
	s.ctx = ctx
}

func notificationEvent(sv Observer, ev Event, re Result) {
	defer func() {
		if err := recover(); err != nil {
			stackInfo := util.GetStack(3)
			slog.Warn("notify event exception", "event_id", ev.ID(), "source", ev.Source(), "destination", ev.Destination(), "panic", err, "stack", stackInfo)

			if re != nil {
				re.Set(nil, cd.NewError(cd.Unexpected, fmt.Sprintf("%v", err)))
			}
		}
	}()

	sv.Notify(ev, re)
}

func NewHub(capacitySize int) Hub {
	return NewHubWithOptions(capacitySize)
}

// NewHubWithOptions 创建带可选配置的 Hub，实现更灵活的缓冲和并发控制
func NewHubWithOptions(capacitySize int, opts ...HubOption) Hub {
	hubOpts := defaultHubOptions(capacitySize)
	for _, opt := range opts {
		if opt != nil {
			opt(hubOpts)
		}
	}

	hub := &hubImpl{
		Execute:               execute.NewExecute(hubOpts.workerPoolSize),
		event2Observer:        ID2ObserverMap{},
		hubActionChannel:      make(chan action, hubOpts.hubActionChanSize),
		laneKey2ActionChannel: LaneKey2ActionChannelMap{},
		perLaneChanSize:       hubOpts.perLaneChanSize,
		laneIdleTimeout:       hubOpts.laneIdleTimeout,
		eventMatchCache:       map[string]ObserverList{},
	}
	go hub.run()
	return hub
}

func NewSimpleObserver(id string, hub Hub) SimpleObserver {
	return &simpleObserver{id: id, matchID: id, eventHub: hub, eventID2ObserverFunc: ID2ObserverFuncMap{}}
}

// NewSimpleObserverWithMatchID 允许将观察者的逻辑标识与 destination 匹配模式分离。
func NewSimpleObserverWithMatchID(id, matchID string, hub Hub) SimpleObserver {
	if matchID == "" {
		matchID = id
	}

	return &simpleObserver{id: id, matchID: matchID, eventHub: hub, eventID2ObserverFunc: ID2ObserverFuncMap{}}
}

func MatchValue(pattern, val string) bool {
	pIdx := 0
	pOffset := 0
	pItems := strings.Split(pattern, "/")

	iIdx := 0
	iOffset := 0
	iItems := strings.Split(val, "/")
	for iIdx < len(iItems) {
		iv := iItems[iIdx]
		if pIdx >= len(pItems) {
			return false
		}

		pv := pItems[pIdx]
		if pv == iv {
			pIdx++
			iIdx++
			continue
		}

		if (pv == "+" || pv == ":id") && iv != "" {
			pIdx++
			iIdx++
			continue
		}

		if pv == "#" && iv != "" {
			pOffset++
			if pIdx+pOffset >= len(pItems) {
				return true
			}

			iOffset++
			if iIdx+iOffset >= len(iItems) {
				return false
			}

			for iIdx+iOffset < len(iItems) {
				if pIdx+pOffset >= len(pItems) {
					return false
				}

				pn := pItems[pIdx+pOffset]
				in := iItems[iIdx+iOffset]
				if pn == in {
					pIdx += pOffset + 1
					pOffset = 0
					break
				}
				if pn == "+" || pn == ":id" {
					if pIdx+pOffset+1 >= len(pItems) {
						return true
					}

					pnn := pItems[pIdx+pOffset+1]
					if pnn == in {
						pIdx += pOffset + 2
						pOffset = 0
						break
					}

					if pv != "#" {
						pOffset++
					}
				}

				iOffset++
				continue
			}

			iIdx += iOffset + 1
			iOffset = 0
			if pIdx > iIdx {
				return false
			}

			continue
		}

		return false
	}

	return pIdx == len(pItems)
}

const (
	subscribe   = 1
	unsubscribe = 2
	post        = 3
	send        = 4
	terminate   = 5
)

type action interface {
	Code() int
}

type subscribeData struct {
	eventID  string
	observer Observer
	result   chan bool
}

func (s *subscribeData) Code() int {
	return subscribe
}

type unsubscribeData subscribeData

func (s *unsubscribeData) Code() int {
	return unsubscribe
}

type postData struct {
	event Event
}

func (s *postData) Code() int {
	return post
}

type sendData struct {
	event  Event
	result chan Result
}

func (s *sendData) Code() int {
	return send
}

type terminateData struct {
	waitGroup *sync.WaitGroup
	result    chan bool
}

func (s *terminateData) Code() int {
	return terminate
}

func (s actionChannel) run(hubPtr *hubImpl) {
	for {
		actionData, actionOK := <-s
		if !actionOK {
			return
		}

		if hubPtr.handleAction(actionData) {
			return
		}
	}
}

func (s *laneActionChannel) run(hubPtr *hubImpl) {
	if hubPtr.laneIdleTimeout <= 0 {
		for {
			actionData, actionOK := <-s.ch
			if !actionOK {
				return
			}
			if hubPtr.handleAction(actionData) {
				return
			}
		}
	}

	timer := time.NewTimer(hubPtr.laneIdleTimeout)
	defer timer.Stop()
	for {
		select {
		case actionData, actionOK := <-s.ch:
			if !actionOK {
				return
			}
			if hubPtr.handleAction(actionData) {
				return
			}
			s.touch()
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(hubPtr.laneIdleTimeout)
		case <-timer.C:
			if s.retireIfIdle(hubPtr) {
				return
			}
			timer.Reset(hubPtr.laneIdleTimeout)
		}
	}
}

type hubImpl struct {
	execute.Execute
	event2ObserverlLock sync.RWMutex
	event2Observer      ID2ObserverMap
	eventMatchCacheLock sync.RWMutex

	hubActionChannel      actionChannel
	laneKey2ChannelLock   sync.RWMutex
	laneKey2ActionChannel LaneKey2ActionChannelMap

	perLaneChanSize int
	laneIdleTimeout time.Duration

	// eventMatchCache 以 eventID 为 key 缓存匹配到的 ObserverList
	// 仅作为加速读路径使用，订阅关系变更时整体失效
	eventMatchCache map[string]ObserverList

	terminateFlag atomic.Bool
}

func newLaneActionChannel(key string, size int) *laneActionChannel {
	if size <= 0 {
		size = 1
	}

	ret := &laneActionChannel{
		key: key,
		ch:  make(actionChannel, size),
	}
	ret.touch()
	return ret
}

func (s *laneActionChannel) touch() {
	s.lastActive.Store(time.Now().UnixNano())
}

func (s *laneActionChannel) enqueue(actionData action, timeout time.Duration) laneEnqueueResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return laneEnqueueClosed
	}

	select {
	case s.ch <- actionData:
		s.touch()
		return laneEnqueueOK
	case <-time.After(timeout):
		return laneEnqueueTimeout
	}
}

func (s *laneActionChannel) retireIfIdle(hubPtr *hubImpl) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return true
	}
	if len(s.ch) > 0 {
		return false
	}
	if time.Since(time.Unix(0, s.lastActive.Load())) < hubPtr.laneIdleTimeout {
		return false
	}

	s.closed = true
	hubPtr.laneKey2ChannelLock.Lock()
	if currentPtr, currentOK := hubPtr.laneKey2ActionChannel[s.key]; currentOK && currentPtr == s {
		delete(hubPtr.laneKey2ActionChannel, s.key)
	}
	hubPtr.laneKey2ChannelLock.Unlock()

	close(s.ch)
	return true
}

func (s *laneActionChannel) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	s.closed = true
	close(s.ch)
}

func eventLaneKey(ev Event) string {
	if ev == nil {
		return ""
	}

	laneKey := ev.LaneKey()
	if laneKey != "" {
		return laneKey
	}

	return ev.Destination()
}

func eventWithLaneContext(ev Event) Event {
	if ev == nil {
		return nil
	}

	return &laneContextEvent{
		Event: ev,
		ctx:   context.WithValue(ev.Context(), laneExecutionContextKey{}, eventLaneKey(ev)),
	}
}

func isReentrantLaneExecution(ev Event, laneKey string) bool {
	if ev == nil || laneKey == "" {
		return false
	}

	currentLaneKey, ok := ev.Context().Value(laneExecutionContextKey{}).(string)
	return ok && currentLaneKey == laneKey
}

func (s *hubImpl) getOrCreateLaneActionChannel(laneKey string) *laneActionChannel {
	s.laneKey2ChannelLock.Lock()
	defer s.laneKey2ChannelLock.Unlock()

	channelVal, channelOK := s.laneKey2ActionChannel[laneKey]
	if channelOK {
		return channelVal
	}

	channelVal = newLaneActionChannel(laneKey, s.perLaneChanSize)
	go channelVal.run(s)
	s.laneKey2ActionChannel[laneKey] = channelVal
	return channelVal
}

func (s *hubImpl) handleAction(actionData action) bool {
	// 同一个 lane 上的所有操作都顺序执行。
	// 包括 post 和 send 操作都需要顺序执行，以保证 lane 内事件顺序。
	switch actionData.Code() {
	case subscribe:
		data := actionData.(*subscribeData)
		s.subscribeInternal(data.eventID, data.observer)
		select {
		case data.result <- true:
			// 成功发送
		case <-time.After(10 * time.Millisecond):
			slog.Warn("timeout sending subscribe result")
		}
	case unsubscribe:
		data := actionData.(*unsubscribeData)
		s.unsubscribeInternal(data.eventID, data.observer)
		select {
		case data.result <- true:
			// 成功发送
		case <-time.After(10 * time.Millisecond):
			slog.Warn("timeout sending unsubscribe result")
		}
	case post:
		data := actionData.(*postData)
		s.postInternal(eventWithLaneContext(data.event))
	case send:
		data := actionData.(*sendData)
		eventWithContext := eventWithLaneContext(data.event)
		result := NewResult(data.event.ID(), data.event.Source(), data.event.Destination())
		s.sendInternal(eventWithContext, result)
		select {
		case data.result <- result:
			// 成功发送
		case <-time.After(100 * time.Millisecond):
			slog.Warn("timeout sending result")
		}
	case terminate:
		data := actionData.(*terminateData)
		data.waitGroup.Done()
		select {
		case data.result <- true:
			// 成功发送
		default:
			// 非阻塞发送，避免在 channel 关闭时 panic
		}
		return true
	default:
		slog.Error("unknown action code", "code", actionData.Code())
	}

	return false
}

func (s *hubImpl) Subscribe(eventID string, observer Observer) {
	if s.terminateFlag.Load() {
		return
	}

	replay := make(chan bool, 1)
	s.Run(func() {
		select {
		case s.hubActionChannel <- &subscribeData{eventID: eventID, observer: observer, result: replay}:
			// 成功发送
		case <-time.After(10 * time.Millisecond):
			// 超时，跳过发送，避免长期阻塞调用方
			replay <- false
		}
	})
	<-replay
	close(replay)
}

func (s *hubImpl) Unsubscribe(eventID string, observer Observer) {
	if s.terminateFlag.Load() {
		return
	}

	replay := make(chan bool, 1)

	s.Run(func() {
		select {
		case s.hubActionChannel <- &unsubscribeData{eventID: eventID, observer: observer, result: replay}:
			// 成功发送
		case <-time.After(10 * time.Millisecond):
			// 超时，跳过发送，避免长期阻塞调用方
			replay <- false
		}
	})
	<-replay
	close(replay)
}

func (s *hubImpl) Post(ev Event) {
	if s.terminateFlag.Load() {
		return
	}
	if ev == nil {
		return
	}

	laneKey := eventLaneKey(ev)
	if isReentrantLaneExecution(ev, laneKey) {
		s.postInternal(eventWithLaneContext(ev))
		return
	}

	actionData := &postData{event: ev}
	for {
		laneChannel := s.getOrCreateLaneActionChannel(laneKey)

		// 再次检查 terminateFlag，防止竞态条件
		if s.terminateFlag.Load() {
			return
		}

		switch laneChannel.enqueue(actionData, 100*time.Millisecond) {
		case laneEnqueueOK:
			return
		case laneEnqueueClosed:
			continue
		case laneEnqueueTimeout:
			slog.Warn("timeout sending post data to channel")
			return
		}
	}
}

func (s *hubImpl) Send(ev Event) (ret Result) {
	if s.terminateFlag.Load() {
		return
	}
	if ev == nil {
		result := NewResult("", "", "")
		result.Set(nil, cd.NewError(cd.IllegalParam, "event is nil"))
		return result
	}

	replay := make(chan Result, 1)
	defer close(replay)

	laneKey := eventLaneKey(ev)
	if isReentrantLaneExecution(ev, laneKey) {
		eventWithContext := eventWithLaneContext(ev)
		result := NewResult(ev.ID(), ev.Source(), ev.Destination())
		s.sendInternal(eventWithContext, result)
		ret = result
		return
	}

	actionData := &sendData{event: ev, result: replay}
	for {
		laneChannel := s.getOrCreateLaneActionChannel(laneKey)

		// 再次检查 terminateFlag，防止竞态条件
		if s.terminateFlag.Load() {
			return
		}

		switch laneChannel.enqueue(actionData, 100*time.Millisecond) {
		case laneEnqueueOK:
			ret = <-replay
			return
		case laneEnqueueClosed:
			continue
		case laneEnqueueTimeout:
			slog.Warn("timeout sending data to channel")
			timeoutResult := NewResult(ev.ID(), ev.Source(), ev.Destination())
			timeoutResult.Set(nil, cd.NewError(cd.Timeout, "send timeout"))
			replay <- timeoutResult
			ret = <-replay
			return
		}
	}
}

func (s *hubImpl) Terminate(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !s.terminateFlag.CompareAndSwap(false, true) {
		return
	}

	var waitGroup sync.WaitGroup
	actionData := &terminateData{result: make(chan bool, 1), waitGroup: &waitGroup}

	s.laneKey2ChannelLock.RLock()
	laneChannels := make([]*laneActionChannel, 0, len(s.laneKey2ActionChannel))
	for _, val := range s.laneKey2ActionChannel {
		laneChannels = append(laneChannels, val)
	}
	s.laneKey2ChannelLock.RUnlock()

	// 先发送终止信号到所有 lane channel。终止不再使用无界等待，
	// 调用方传入的 shutdown context 是唯一等待预算。
	for _, val := range laneChannels {
		waitGroup.Add(1)
		switch val.enqueue(actionData, 100*time.Millisecond) {
		case laneEnqueueOK:
			// 成功发送
		default:
			waitGroup.Done()
		}
	}

	waitGroup.Add(1)
	select {
	case s.hubActionChannel <- actionData:
		// 成功发送
	case <-ctx.Done():
		waitGroup.Done()
	}

	if !waitGroupContext(&waitGroup, ctx) {
		slog.Warn("hub action channels did not terminate before context done", "err", ctx.Err())
	}

	// 等待所有 Execute.Run 提交的任务完成
	if !s.WaitContext(ctx) {
		slog.Warn("hub execute tasks did not drain before context done", "err", ctx.Err())
	}

	// 等待所有 goroutine 完成处理
	s.laneKey2ChannelLock.Lock()
	laneChannels = []*laneActionChannel{}
	for _, val := range s.laneKey2ActionChannel {
		laneChannels = append(laneChannels, val)
	}
	s.laneKey2ActionChannel = LaneKey2ActionChannelMap{}
	s.laneKey2ChannelLock.Unlock()
	for _, val := range laneChannels {
		val.close()
	}
	s.event2Observer = ID2ObserverMap{}
}

func waitGroupContext(waitGroup *sync.WaitGroup, ctx context.Context) bool {
	done := make(chan struct{})
	go func() {
		defer close(done)
		waitGroup.Wait()
	}()

	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *hubImpl) run() {
	s.hubActionChannel.run(s)

	close(s.hubActionChannel)
}

func (s *hubImpl) subscribeInternal(eventID string, observer Observer) {
	s.event2ObserverlLock.Lock()
	defer s.event2ObserverlLock.Unlock()

	observerList, observerOK := s.event2Observer[eventID]
	if !observerOK {
		observerList = ObserverList{}
	}
	existFlag := false
	for _, val := range observerList {
		if val.ID() == observer.ID() {
			existFlag = true
			break
		}
	}
	if !existFlag {
		observerList = append(observerList, observer)
	}
	s.event2Observer[eventID] = observerList

	// 订阅关系变更，清空匹配缓存
	s.eventMatchCacheLock.Lock()
	s.eventMatchCache = map[string]ObserverList{}
	s.eventMatchCacheLock.Unlock()
}

func (s *hubImpl) unsubscribeInternal(eventID string, observer Observer) {
	s.event2ObserverlLock.Lock()
	defer s.event2ObserverlLock.Unlock()

	observerList, observerOK := s.event2Observer[eventID]
	if !observerOK {
		return
	}

	newList := ObserverList{}
	for _, val := range observerList {
		if val.ID() == observer.ID() {
			continue
		}

		newList = append(newList, val)
	}
	if len(newList) > 0 {
		s.event2Observer[eventID] = newList
	} else {
		delete(s.event2Observer, eventID)
	}

	// 订阅关系变更，清空匹配缓存
	s.eventMatchCacheLock.Lock()
	s.eventMatchCache = map[string]ObserverList{}
	s.eventMatchCacheLock.Unlock()
}

func (s *hubImpl) postInternal(ev Event) {
	matchList := make(ObserverList, 0, 4)
	cacheKey := matchCacheKey(ev.ID(), ev.Destination())
	if cached, ok := s.getCachedObservers(cacheKey); ok {
		matchList = append(matchList, cached...)
	} else {
		tmpMatch := s.findMatchingObservers(ev)
		matchList = append(matchList, tmpMatch...)
		s.setCachedObservers(cacheKey, tmpMatch)
	}

	for _, sv := range matchList {
		notificationEvent(sv, ev, nil)
	}
}

func (s *hubImpl) sendInternal(ev Event, re Result) {
	matchList := make(ObserverList, 0, 4)
	finalFlag := false
	cacheKey := matchCacheKey(ev.ID(), ev.Destination())
	if cached, ok := s.getCachedObservers(cacheKey); ok {
		matchList = append(matchList, cached...)
		finalFlag = len(cached) > 0
	} else {
		tmpMatch := s.findMatchingObservers(ev)
		matchList = append(matchList, tmpMatch...)
		finalFlag = len(tmpMatch) > 0
		s.setCachedObservers(cacheKey, tmpMatch)
	}

	for _, sv := range matchList {
		notificationEvent(sv, ev, re)
	}

	if !finalFlag && re != nil {
		re.Set(nil, cd.NewError(cd.Unexpected, fmt.Sprintf("missing observer, event:[id-%v, source-%s, destination-%s]", ev.ID(), ev.Source(), ev.Destination())))
	}
}

func matchCacheKey(eventID, destination string) string {
	return eventID + "\x00" + destination
}

func (s *hubImpl) getCachedObservers(cacheKey string) (ObserverList, bool) {
	s.eventMatchCacheLock.RLock()
	defer s.eventMatchCacheLock.RUnlock()

	cached, ok := s.eventMatchCache[cacheKey]
	if !ok {
		return nil, false
	}

	result := make(ObserverList, len(cached))
	copy(result, cached)
	return result, true
}

func (s *hubImpl) setCachedObservers(cacheKey string, observers ObserverList) {
	s.eventMatchCacheLock.Lock()
	defer s.eventMatchCacheLock.Unlock()

	if s.eventMatchCache == nil {
		s.eventMatchCache = map[string]ObserverList{}
	}
	result := make(ObserverList, len(observers))
	copy(result, observers)
	s.eventMatchCache[cacheKey] = result
}

func (s *hubImpl) findMatchingObservers(ev Event) ObserverList {
	matchList := make(ObserverList, 0, 4)

	s.event2ObserverlLock.RLock()
	defer s.event2ObserverlLock.RUnlock()

	for key, value := range s.event2Observer {
		if MatchValue(key, ev.ID()) {
			for _, sv := range value {
				if matchDestination(ev.Destination(), sv.ID()) {
					matchList = append(matchList, sv)
				}
			}
		}
	}

	return matchList
}

func matchDestination(destination, observerID string) bool {
	return MatchValue(destination, observerID) || MatchValue(observerID, destination)
}

type simpleObserver struct {
	id                   string
	matchID              string
	eventHub             Hub
	eventID2ObserverFunc ID2ObserverFuncMap
	eventIDLock          sync.RWMutex
}

func (s *simpleObserver) ID() string {
	if s.matchID != "" {
		return s.matchID
	}

	return s.id
}

func (s *simpleObserver) Notify(ev Event, re Result) {
	var funcVal ObserverFunc
	func() {
		s.eventIDLock.RLock()
		defer s.eventIDLock.RUnlock()

		for k, v := range s.eventID2ObserverFunc {
			if ev.Match(k) {
				funcVal = v
				break
			}
		}
	}()

	if funcVal != nil {
		func() {
			defer func() {
				if err := recover(); err != nil {
					stackInfo := util.GetStack(3)
					slog.Warn("notify event exception", "event_id", ev.ID(), "source", ev.Source(), "destination", ev.Destination(), "panic", err, "stack", stackInfo)

					if re != nil {
						re.Set(nil, cd.NewError(cd.Unexpected, fmt.Sprintf("%v", err)))
					}
				}
			}()

			funcVal(ev, re)
		}()
	}
}

func (s *simpleObserver) Subscribe(eventID string, observerFunc ObserverFunc) {
	okFlag := false
	func() {
		s.eventIDLock.Lock()
		defer s.eventIDLock.Unlock()

		_, ok := s.eventID2ObserverFunc[eventID]
		if ok {
			slog.Warn("duplicate eventID", "value", eventID)
			return
		}

		s.eventID2ObserverFunc[eventID] = observerFunc
		okFlag = true
	}()

	if okFlag {
		s.eventHub.Subscribe(eventID, s)
	}
}

func (s *simpleObserver) Unsubscribe(eventID string) {
	okFlag := false
	func() {
		s.eventIDLock.Lock()
		defer s.eventIDLock.Unlock()

		_, ok := s.eventID2ObserverFunc[eventID]
		if !ok {
			slog.Warn("not exist eventID", "value", eventID)
			return
		}

		delete(s.eventID2ObserverFunc, eventID)
		okFlag = true
	}()

	if okFlag {
		s.eventHub.Unsubscribe(eventID, s)
	}
}
