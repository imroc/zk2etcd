package sync

import (
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/diff"
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/atomic"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Syncer struct {
	zkPrefix         []string
	zkExcludePrefix  []string
	m                sync.Map
	concurrency      uint
	keysToSync       chan string
	keyCountSynced   atomic.Int32
	fullSyncInterval time.Duration
	watcher          map[string]struct{}
	watcherLock      sync.Mutex
	stop             <-chan struct{}
}

func New(zkPrefix, zkExcludePrefix []string, concurrency uint, fullSyncInterval time.Duration, stop <-chan struct{}) *Syncer {
	return &Syncer{
		zkPrefix:         zkPrefix,
		zkExcludePrefix:  zkExcludePrefix,
		concurrency:      concurrency,
		keysToSync:       make(chan string, concurrency),
		fullSyncInterval: fullSyncInterval,
		watcher:          make(map[string]struct{}),
		stop:             stop,
	}
}

func (s *Syncer) startWorker() {
	log.Info("start worker",
		"concurrency", s.concurrency,
	)

	for i := uint(0); i < s.concurrency; i++ {
		go func() {
			for {
				key := <-s.keysToSync
				s.syncKey(key)
			}
		}()
	}
}

func (s *Syncer) SyncIncremental() {
	log.Info("start incremental sync")

	for _, prefix := range s.zkPrefix {
		s.syncWatchRecursive(prefix, s.stop)
	}
}

func (s *Syncer) FullSync() {
	log.Info("start full sync")
	before := time.Now()

	d := diff.New(s.zkPrefix, s.zkExcludePrefix, s.concurrency)

	log.Info("start full sync diff")
	d.Run()
	log.Info("complete full sync diff")

	log.Info("start full sync fix")
	d.Fix()
	log.Info("complete full sync fix")

	cost := time.Since(before)
	log.Infow("full sync completed",
		"cost", cost.String(),
	)
}

func (s *Syncer) ensureClients() {
	log.Debugw("check zk")
	for _, prefix := range s.zkPrefix {
		zookeeper.EnsureExists(prefix)
	}
	log.Info("check zk success")
	log.Debugw("check etcd")
	etcd.Get("/")
	log.Info("check etcd success")
}

func (s *Syncer) startHttpServer() {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":80", nil)
	if err != nil {
		log.Fatalw("http listen failed",
			"error", err.Error(),
		)
	}
}

// Run until a signal is received, this function won't block
func (s *Syncer) Run() {

	// 启动 metrics server
	go s.startHttpServer()

	// 检查 zk 和 etcd
	s.ensureClients()

	// 启动 worker
	s.startWorker()

	// 全量同步一次
	s.FullSync()

	// 定期全量同步
	s.StartFullSyncInterval()

	// 继续同步增量
	s.SyncIncremental()

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

func (s *Syncer) removeWatch(key string) {
	s.watcherLock.Lock()
	_, ok := s.watcher[key]
	if !ok {
		log.Warnw("remove watch but watcher does not exist",
			"key", key,
		)
		s.watcherLock.Unlock()
		return
	}
	delete(s.watcher, key)
	s.watcherLock.Unlock()
}

func (s *Syncer) handleEvent(event zk.Event) bool {
	log.Debugw("handle event",
		"type", event.Type.String(),
		"state", event.State.String(),
		"error", event.Err,
		"path", event.Path,
	)
	switch event.Type {
	case zk.EventNodeDeleted:
		etcd.DeleteWithPrefix(event.Path)
		s.removeWatch(event.Path)
		return false
	case zk.EventNodeChildrenChanged:
		s.syncChildren(event.Path)
		return true
	case zk.EventNotWatching:
		log.Warnw("received zk not watching event",
			"type", event.Type.String(),
			"state", event.State.String(),
			"error", event.Err,
			"path", event.Path,
		)
		zookeeper.ReConnect()
		return true
	default:
		log.Warnw("unknown event",
			"type", event.Type.String(),
			"state", event.State.String(),
			"error", event.Err,
			"path", event.Path,
		)
		return false
	}
}

func (s *Syncer) watch(key string, stop <-chan struct{}) []string {
	s.watcherLock.Lock()
	if _, ok := s.watcher[key]; ok {
		s.watcherLock.Unlock()
		return nil
	}
	s.watcher[key] = struct{}{}
	s.watcherLock.Unlock()

	log.Debugw("watch key",
		"key", key,
	)

	children, ch := zookeeper.ListW(key)

	if ch == nil {
		log.Warnw("key not exist",
			"key", key,
		)
		return children
	}

	go func() {
		for {
			select {
			case event := <-ch:
				shouldContinue := s.handleEvent(event)
				if !shouldContinue {
					return
				}
				children, ch = zookeeper.ListW(key)
				if ch == nil {
					log.Debugw("continue list watch children but key not exist any more",
						"key", key,
					)
					s.removeWatch(event.Path)
					return
				}
				for _, child := range children {
					newKey := filepath.Join(key, child)
					s.watch(newKey, stop)
					s.syncKey(newKey)
				}
			case <-stop:
				return
			}
		}
	}()

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
			log.Debugw("ignore key in excluded prefix",
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
	children := zookeeper.List(key)

	for _, child := range children {
		fullPath := filepath.Join(key, child)
		s.syncKeyRecursive(fullPath)
	}
}

func (s *Syncer) syncKey(key string) {
	log.Debugw("sync key",
		"key", key,
	)
	defer s.keyCountSynced.Inc()
	zkValue, existInZK := zookeeper.Get(key)
	etcdValue, existInEtcd := etcd.Get(key)
	switch {
	case existInZK && !existInEtcd: // etcd 中缺失，补齐
		log.Debugw("key not exist in etcd, put in etcd",
			"key", key,
			"value", zkValue,
		)
		etcd.Put(key, zkValue)
		s.syncWatchRecursive(key, s.stop) // 可能是新增的，确保 watch 下
	case !existInZK && existInEtcd: // etcd 中多出 key，删除
		log.Debugw("key not exist in zk, remove in etcd",
			"key", key,
		)
		etcd.DeleteWithPrefix(key)
	case existInZK && existInEtcd && (zkValue != etcdValue): // key 都存在，但 value 不同，纠正 etcd 中的 value
		log.Debugw("value differs",
			"key", key,
			"zkValue", zkValue,
			"etcdValue", etcdValue,
		)
		etcd.Put(key, zkValue)
	default:
		log.Debugw("sync ignore",
			"key", key,
			"existInZK", existInZK,
			"existInEtcd", existInEtcd,
			"zkValue", zkValue,
			"etcdValue", etcdValue,
		)
	}
}

func (s *Syncer) syncChildren(key string) {
	log.Infow("sync children",
		"key", key,
	)
	children := zookeeper.List(key)
	for _, child := range children {
		fullPath := filepath.Join(key, child)
		s.syncKey(fullPath)
	}
}
