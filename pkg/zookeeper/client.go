package zookeeper

import (
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/log"
	"time"
)

type Client struct {
	*log.Logger
	servers   []string
	conn      *zk.Conn
	watchConn *zk.Conn
	callback  zk.EventCallback
}

func NewClient(logger *log.Logger, servers []string) *Client {
	client := &Client{
		Logger:  logger,
		servers: servers,
	}
	return client
}

func (c *Client) SetCallback(callback zk.EventCallback) {
	c.callback = callback
}

func (c *Client) EnsureExists(key string) {
	// check path exsitence
	exists := false
	for !exists {
		var err error
		exists, _, err = c.getConn().Exists(key)
		if err != nil {
			c.Errorw("failed to check path existence",
				"error", err,
				"key", key,
			)
		}
		if !exists {
			c.Warnw("zookeeper path doesn't exist, wait until it's created",
				"key", key,
			)
		}
		time.Sleep(time.Second * 2)
	}
}

func (c *Client) getConn() *zk.Conn {
	if c.conn == nil {
		c.conn = c.connectUntilSuccess(false)
	}
	return c.conn
}

func (c *Client) getWatchConn() *zk.Conn {
	if c.watchConn == nil {
		c.watchConn = c.connectUntilSuccess(true)
	}
	return c.watchConn
}

func (c *Client) Get(key string) string {
	value, _, err := c.getConn().Get(key)
	if err != nil {
		c.Errorw("zk get failed",
			"key", key,
			"error", err,
		)
		return ""
	}
	c.Debugw("zk get",
		"key", key,
		"value", value,
	)
	return string(value)
}

// Delete 暂时不用
func (c *Client) Delete(key string) {
	_, s, err := c.getConn().Get(key)
	if err != nil {
		c.Errorw("zk delete failed",
			"key", key,
			"error", err,
		)
		return
	}
	// TODO: 提升健壮性，处理删除冲突，进行重试
	err = c.getConn().Delete(key, s.Version)
	if err != nil {
		c.Errorw("zk delete failed",
			"key", key,
			"error", err,
		)
	}
}

func (c *Client) List(key string) []string {
	children, _, err := c.getConn().Children(key)
	if err != nil {
		c.Errorw("zk list failed",
			"key", key,
			"error", err,
		)
		return nil
	}
	c.Debugw("zk list",
		"key", key,
		"children", children,
	)
	return children
}

func (c *Client) ListW(key string) (children []string, ch <-chan zk.Event) {
	children, _, ch, err := c.getWatchConn().ChildrenW(key)
	for err != nil && err != zk.ErrNoNode {
		c.Errorw("failed to list watch zookeeper",
			"key", key,
			"error", err,
		)
		time.Sleep(1 * time.Second)
		children, _, ch, err = c.getWatchConn().ChildrenW(key)
	}
	c.Debugw("zk list watch",
		"key", key,
		"children", children,
	)
	return
}

func (c *Client) connect(needCallback bool) (conn *zk.Conn, err error) {
	c.Debugw("zk connect",
		"servers", c.servers,
	)
	if needCallback {
		option := zk.WithEventCallback(c.callback)
		conn, _, err = zk.Connect(c.servers, time.Second*10, option)
	} else {
		conn, _, err = zk.Connect(c.servers, time.Second*10)
	}
	return
}

func (c *Client) connectUntilSuccess(needCallback bool) *zk.Conn {
	conn, err := c.connect(needCallback)
	for err != nil { // 如果连接失败，一直重试，直到成功
		c.Errorw("failed to connect to zooKeeper, retrying...",
			"error", err,
			"servers", c.servers,
		)
		time.Sleep(time.Second)
		conn, err = c.connect(needCallback)
	}
	return conn
}
