package controller

import (
	"context"
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Controller struct {
	*log.Logger
	zk         *zookeeper.Client
	zkPrefix   string
	etcdAddr   string
	zkConn     *zk.Conn
	zkConn2    *zk.Conn
	etcdClient *clientv3.Client
	m          sync.Map
}

func New(zkClient *zookeeper.Client, zkPrefix, etcdAddr string, logger *log.Logger) *Controller {
	return &Controller{
		zk:       zkClient,
		zkPrefix: zkPrefix,
		etcdAddr: etcdAddr,
		Logger:   logger,
	}
}

// Run until a signal is received, this function won't block
func (c *Controller) Run(stop <-chan struct{}) {
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   strings.Split(c.etcdAddr, ","),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		panic(err)
	}
	c.etcdClient = etcdClient

	c.zk.SetCallback(c.callback)

	c.zk.EnsureExists(c.zkPrefix)

	// 启动全量同步一次
	c.Info("start full sync")
	c.syncKeyRecursive(c.zkPrefix)
	c.Info("full sync completed")

	// 继续同步增量
	c.Info("start incremental sync")
	c.syncWatch(c.zkPrefix, stop)
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
	c.syncKey(key)
	children := c.zk.List(key)

	for _, child := range children {
		fullPath := filepath.Join(key, child)
		c.syncKeyRecursive(fullPath)
	}
}

func (c *Controller) connectZKUntilSuccess(hosts []string) *zk.Conn {
	option := zk.WithEventCallback(c.callback)
	conn, _, err := zk.Connect(hosts, time.Second*10, option)
	//Keep trying to connect to ZK until succeed
	for err != nil {
		c.Errorw("failed to connect to zooKeeper, retrying...",
			"error", err,
			"hosts", hosts,
		)
		time.Sleep(time.Second)
		conn, _, err = zk.Connect(hosts, time.Second*10, option)
	}
	return conn
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
	etcdValue, exist := c.etcdGet(key)
	if exist { // key 存在
		if zkValue != etcdValue { // 但 value 不同，更新下
			c.etcdPut(key, zkValue)
		}
	} else { // key 不存在，创建一个
		c.etcdPut(key, zkValue)
	}
}

func (c *Controller) syncChildren(key string) {
	c.Debugw("zk sync children",
		"key", key,
	)
	children := c.zk.List(key)
	for _, child := range children {
		fullPath := filepath.Join(key, child)
		c.syncKey(fullPath)
	}
}

func (c *Controller) etcdPut(key, value string) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	_, err := c.etcdClient.Put(ctx, key, value)
	if err != nil {
		c.Errorw("etcd failed to put",
			"key", key,
			"value", value,
			"error", err,
		)
	}
}

func (c *Controller) etcdGet(path string) (string, bool) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	resp, err := c.etcdClient.Get(ctx, path)
	if err != nil {
		c.Errorw("etcd failed to get",
			"key", path,
			"error", err,
		)
		return "", false
	}
	if len(resp.Kvs) != 0 {
		return string(resp.Kvs[0].Value), true
	}
	return "", false
}
