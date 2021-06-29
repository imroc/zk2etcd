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
	zk               *zookeeper.Client
	zkPrefix         []string
	zkExcludePrefix  []string
	etcd             *etcd.Client
	m                sync.Map
	concurrency      uint
	keysToSync       chan string
	keyCountSynced   atomic.Int32
	fullSyncInterval time.Duration
}

func New(zkClient *zookeeper.Client, zkPrefix, zkExcludePrefix []string, etcd *etcd.Client, logger *log.Logger, concurrency uint, fullSyncInterval time.Duration) *Syncer {
	return &Syncer{
		zk:               zkClient,
		zkPrefix:         zkPrefix,
		zkExcludePrefix:  zkExcludePrefix,
		Logger:           logger,
		etcd:             etcd,
		concurrency:      concurrency,
		keysToSync:       make(chan string, concurrency),
		fullSyncInterval: fullSyncInterval,
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

func (s *Syncer) SyncIncremental(stop <-chan struct{}) {
	s.Info("start incremental sync")

	for _, prefix := range s.zkPrefix {
		s.syncWatchRecursive(prefix, stop)
	}
}

func (s *Syncer) FullSync() {
	s.Info("start full sync")
	before := time.Now()

	for _, prefix := range s.zkPrefix {
		s.syncKeyRecursive(prefix)
	}

	cost := time.Since(before)
	s.Infow("full sync completed",
		"cost", cost.String(),
		"keyCount", s.keyCountSynced.String(),
	)
}

func (s *Syncer) ensureClients() {
	s.Debugw("check zk")
	for _, prefix := range s.zkPrefix {
		s.zk.EnsureExists(prefix)
	}
	s.Info("check zk success")
	s.Debugw("check etcd")
	s.etcd.Get("/")
	s.Info("check etcd success")
}

// Run until a signal is received, this function won't block
func (s *Syncer) Run(stop <-chan struct{}) {
	s.zk.SetCallback(s.callback)

	// 检查 zk 和 etcd
	s.ensureClients()

	// 启动 worker
	s.startWorker()

	// 全量同步一次
	s.FullSync()

	// 定期全量同步
	s.StartFullSyncInterval()

	// 继续同步增量
	s.SyncIncremental(stop)

}

func (s *Syncer) StartFullSyncInterval() {
	if s.fullSyncInterval <= 0 {
		return
	}
	go func() {
		for range time.Tick(s.fullSyncInterval) {
			s.FullSync()
		}
	}()
}

func (s *Syncer) watch(key string, stop <-chan struct{}) []string {
	s.m.Store(key, true)
	s.Debugw("before watching children",
		"key", key,
	)
	children, ch := s.zk.ListW(key)
	s.Debugw("after watching children",
		"key", key,
	)
	if ch != nil {
		go func() {
			for {
				select {
				case event := <-ch:
					s.Infow("received event",
						"event", event,
						"parent", key,
					)
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
								s.syncWatchRecursive(filepath.Join(key, child), stop)
							}
						}
					}
				case <-stop:
					return
				}
			}
		}()
	} else {
		s.Warnw("got nil channel from list children watch",
			"key", key,
			"children", children,
		)
	}
	return children
}

func (s *Syncer) syncWatchRecursive(key string, stop <-chan struct{}) {
	if s.shouldExclude(key) {
		return
	}

	// watch children
	children := s.watch(key, stop)

	// watch children recursively
	for _, child := range children {
		s.syncWatchRecursive(filepath.Join(key, child), stop)
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
	zkValue, existInZK := s.zk.Get(key)
	etcdValue, existInEtcd := s.etcd.Get(key)
	switch {
	case existInZK && !existInEtcd: // etcd 中缺失，补齐
		s.Debugw("key not exist in etcd, put in etcd",
			"key", key,
			"value", zkValue,
		)
		s.etcd.Put(key, zkValue)
	case !existInZK && existInEtcd: // etcd 中多出 key，删除
		s.Debugw("key not exist in zk, remove in etcd",
			"key", key,
		)
		s.etcd.Delete(key)
	case existInZK && existInEtcd && (zkValue != etcdValue): // key 都存在，但 value 不同，纠正 etcd 中的 value
		s.Debugw("value differs",
			"key", key,
			"zkValue", zkValue,
			"etcdValue", etcdValue,
		)
		s.etcd.Put(key, zkValue)
	default:
		s.Warnw("syncKey not match",
			"key", key,
			"existInZK", existInZK,
			"existInEtcd", existInEtcd,
			"zkValue", zkValue,
			"etcdValue", etcdValue,
		)
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
