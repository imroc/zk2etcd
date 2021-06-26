package controller

import (
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	"path/filepath"
	"strings"
	"sync"
)

type Controller struct {
	*log.Logger
	zk          *zookeeper.Client
	zkPrefix    string
	etcd        *etcd.Client
	m           sync.Map
	concurrency uint
	keysToSync  chan string
}

func New(zkClient *zookeeper.Client, zkPrefix string, etcd *etcd.Client, logger *log.Logger, concurrency uint) *Controller {
	return &Controller{
		zk:          zkClient,
		zkPrefix:    zkPrefix,
		Logger:      logger,
		etcd:        etcd,
		concurrency: concurrency,
		keysToSync:  make(chan string, concurrency),
	}
}

func (c *Controller) startWorker() {
	c.Info("start worker",
		"concurrency", c.concurrency,
	)
	for i := uint(0); i < c.concurrency; i++ {
		go func() {
			for {
				key := <-c.keysToSync
				c.syncKey(key)
			}
		}()
	}
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	c.zk.SetCallback(c.callback)

	// 检查 zk 和 etcd
	c.Debugw("check zk")
	c.zk.EnsureExists(c.zkPrefix)
	c.Info("check zk success")
	c.Debugw("check etcd")
	c.etcd.Get("/")
	c.Info("check etcd success")

	// 启动 worker
	c.startWorker()

	// 获取 prefix
	prefixes := strings.Split(c.zkPrefix, ",")

	// 全量同步一次
	c.Info("start full sync")
	for _, prefix := range prefixes {
		c.syncKeyRecursive(prefix)
	}
	c.Info("full sync completed")

	// 继续同步增量
	c.Info("start incremental sync")
	for _, prefix := range prefixes {
		c.syncWatch(prefix, stop)
	}
}

func (c *Controller) syncWatch(key string, stop <-chan struct{}) {
	c.m.Store(key, true)
	children, ch := c.zk.ListW(key)

	if ch != nil {
		go func() {
			for {
				select {
				case event := <-ch:
					if event.Type == zk.EventNodeDeleted {
						c.m.Delete(key)
						c.etcd.Delete(key)
						return
					}
					children, ch = c.zk.ListW(key)
					if ch == nil {
						c.Debugw("stop watch",
							"key", key,
						)
						return
					} else {
						for _, child := range children {
							_, ok := c.m.Load(filepath.Join(key, child))
							if !ok {
								c.syncWatch(filepath.Join(key, child), stop)
							}
						}
					}
				case <-stop:
					return
				}
			}
		}()
	}

	for _, child := range children {
		c.syncWatch(filepath.Join(key, child), stop)
	}
}

func (c *Controller) syncKeyRecursive(key string) {
	c.keysToSync <- key
	children := c.zk.List(key)

	for _, child := range children {
		fullPath := filepath.Join(key, child)
		c.syncKeyRecursive(fullPath)
	}
}

func (c *Controller) callback(event zk.Event) {
	switch event.Type {
	case zk.EventNodeChildrenChanged:
		c.syncChildren(event.Path)
	default:
		c.Debugw("ignore event",
			"event", event,
		)
	}
}

func (c *Controller) syncKey(key string) {
	c.Debugw("sync key",
		"key", key,
	)
	zkValue := c.zk.Get(key)
	etcdValue, exist := c.etcd.Get(key)
	if exist { // key 存在
		if zkValue != etcdValue { // 但 value 不同，更新下
			c.etcd.Put(key, zkValue)
		}
		// key与value相同，忽略
	} else { // key 不存在，创建一个
		c.etcd.Put(key, zkValue)
	}
}

func (c *Controller) syncChildren(key string) {
	c.Debugw("sync children",
		"key", key,
	)
	children := c.zk.List(key)
	for _, child := range children {
		fullPath := filepath.Join(key, child)
		c.syncKey(fullPath)
	}
}
