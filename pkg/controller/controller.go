package controller

import (
	"context"
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"path/filepath"
	"strings"
	"time"
)

type Controller struct {
	*log.Logger
	zkAddr     string
	zkPrefix   string
	etcdAddr   string
	zkConn     *zk.Conn
	zkConn2    *zk.Conn
	etcdClient *clientv3.Client
}

func New(zkAddr, zkPrefix, etcdAddr string, logger *log.Logger) *Controller {
	return &Controller{
		zkAddr:   zkAddr,
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

	hosts := strings.Split(c.zkAddr, ",")
	conn := c.connectZKUntilSuccess(hosts)
	conn2 := c.connectZKUntilSuccess(hosts)
	c.zkConn = conn
	c.zkConn2 = conn2

	// check path exsitence
	exists := false
	for !exists {
		var err error
		exists, _, err = conn.Exists(c.zkPrefix)
		if err != nil {
			c.Errorw("failed to check path existence",
				"error", err,
				"path", c.zkPrefix,
			)
		}
		if !exists {
			c.Warnw("zookeeper path doesn't exist, wait until it's created",
				"path", c.zkPrefix,
			)
		}
		time.Sleep(time.Second * 2)
	}

	// 启动全量同步一次
	c.syncAll(c.zkPrefix)
	c.Info("full sync completed")

	// 继续同步增量
	c.Info("start incremental sync")
	w := zookeeper.New(conn, c.zkPrefix)
	w.Run(stop)
}

func (c *Controller) syncAll(path string) {
	c.Debugw("syncing all",
		"node", path,
	)
	c.syncPath(path)
	children, _, err := c.zkConn2.Children(path)
	if err != nil {
		c.Errorw("zk failed to get children",
			"node", path,
			"error", err,
		)
		return
	}
	for _, child := range children {
		fullPath := filepath.Join(path, child)
		c.syncAll(fullPath)
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

func (c *Controller) syncPath(path string) {
	v, _, err := c.zkConn2.Get(path)
	if err != nil {
		c.Errorw("failed to get node",
			"node", path,
			"error", err,
		)
		return
	}
	zkValue := string(v)
	etcdValue, exist := c.etcdGet(path)
	if exist { // key 存在
		if zkValue != etcdValue { // 但 value 不同，更新下
			c.etcdPut(path, zkValue)
		}
	} else { // key 不存在，创建一个
		c.etcdPut(path, zkValue)
	}
}

func (c *Controller) syncChildren(path string) {
	c.Debugw("zk sync children",
		"node", path,
	)
	children, _, err := c.zkConn2.Children(path)
	c.Debugw("zk got children",
		"children", children,
	)
	if err != nil {
		c.Errorw("zk failed to get children",
			"node", path,
			"error", err,
		)
		return
	}
	for _, child := range children {
		fullPath := filepath.Join(path, child)
		c.syncPath(fullPath)
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
