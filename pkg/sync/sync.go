package sync

import (
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	"go.uber.org/atomic"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Syncer struct {
	*log.Logger
	zk              *zookeeper.Client
	zkPrefix        []string
	zkExcludePrefix []string
	etcd            *etcd.Client
	m               sync.Map
	concurrency     uint
	keysToSync      chan string
	keyCountSynced  atomic.Int32
}

func New(zkClient *zookeeper.Client, zkPrefix, zkExcludePrefix []string, etcd *etcd.Client, logger *log.Logger, concurrency uint) *Syncer {
	return &Syncer{
		zk:              zkClient,
		zkPrefix:        zkPrefix,
		zkExcludePrefix: zkExcludePrefix,
		Logger:          logger,
		etcd:            etcd,
		concurrency:     concurrency,
		keysToSync:      make(chan string, concurrency),
	}
}

func (s *Syncer) startWorker() {
	s.Info("start worker",
		"concurrency", s.concurrency,
	)
	go func() {
		for range time.Tick(1 * time.Second) {
			s.Infow("synced key count",
				"count", s.keyCountSynced.String(),
			)
		}
	}()
	for i := uint(0); i < s.concurrency; i++ {
		go func() {
			for {
				key := <-s.keysToSync
				s.syncKey(key)
			}
		}()
	}
}

// Run until a signal is received, this function won't block
func (s *Syncer) Run(stop <-chan struct{}) {
	s.zk.SetCallback(s.callback)

	// 检查 zk 和 etcd
	s.Debugw("check zk")
	for _, prefix := range s.zkPrefix {
		s.zk.EnsureExists(prefix)
	}
	s.Info("check zk success")
	s.Debugw("check etcd")
	s.etcd.Get("/")
	s.Info("check etcd success")

	// 启动 worker
	s.startWorker()

	// 全量同步一次
	s.Info("start full sync")
	before := time.Now()
	for _, prefix := range s.zkPrefix {
		s.syncKeyRecursive(prefix)
	}
	cost := time.Since(before)
	s.Infow("full sync completed",
		"cost", cost.String(),
		"keyCount", s.keyCountSynced,
	)

	// 继续同步增量
	s.Info("start incremental sync")
	for _, prefix := range s.zkPrefix {
		s.syncWatch(prefix, stop)
	}
}

func (s *Syncer) watch(key string, stop <-chan struct{}) []string {
	s.m.Store(key, true)
	children, ch := s.zk.ListW(key)
	if ch != nil {
		go func() {
			for {
				select {
				case event := <-ch:
					if event.Type == zk.EventNodeDeleted {
						s.m.Delete(key)
						s.etcd.Delete(key)
						return
					}
					children, ch = s.zk.ListW(key)
					if ch == nil {
						s.Debugw("stop watch",
							"key", key,
						)
						return
					} else {
						for _, child := range children {
							_, ok := s.m.Load(filepath.Join(key, child))
							if !ok {
								s.syncWatch(filepath.Join(key, child), stop)
							}
						}
					}
				case <-stop:
					return
				}
			}
		}()
	}
	return children
}

func (s *Syncer) syncWatch(key string, stop <-chan struct{}) {
	if s.shouldExclude(key) {
		return
	}

	// watch children
	children := s.watch(key, stop)

	// watch children recursively
	for _, child := range children {
		s.syncWatch(filepath.Join(key, child), stop)
	}
}

func (s *Syncer) shouldExclude(key string) bool {
	for _, prefix := range s.zkExcludePrefix { // exclude prefix
		if strings.HasPrefix(key, prefix) {
			s.Debugw("ignore key in excluded prefix",
				"key", key,
				"excludePrefix", prefix,
			)
			return true
		}
	}
	return false
}

func (s *Syncer) syncKeyRecursive(key string) {
	if s.shouldExclude(key) {
		return
	}

	s.keysToSync <- key
	children := s.zk.List(key)

	for _, child := range children {
		fullPath := filepath.Join(key, child)
		s.syncKeyRecursive(fullPath)
	}
}

func (s *Syncer) callback(event zk.Event) {
	switch event.Type {
	case zk.EventNodeChildrenChanged:
		s.syncChildren(event.Path)
	default:
		s.Debugw("ignore event",
			"event", event,
		)
	}
}

func (s *Syncer) syncKey(key string) {
	s.Debugw("sync key",
		"key", key,
	)
	defer s.keyCountSynced.Inc()
	zkValue, ok := s.zk.Get(key)
	etcdValue, exist := s.etcd.Get(key)
	if exist { // etcd 中 key 存在
		if !ok { // zk 中 key 不存在，删除 etcd 中对应 key
			s.etcd.Delete(key)
			return
		}
		if zkValue != etcdValue { // zk 与 etcd 的 key 都存在，但 value 不同，更新下 etcd
			s.etcd.Put(key, zkValue)
		}
		// key与value相同，忽略
	} else {    // etcd key 不存在
		if ok { // zk key 存在，etcd 补数据
			s.etcd.Put(key, zkValue)
		}
	}
}

func (s *Syncer) syncChildren(key string) {
	s.Debugw("sync children",
		"key", key,
	)
	children := s.zk.List(key)
	for _, child := range children {
		fullPath := filepath.Join(key, child)
		s.syncKey(fullPath)
	}
}
