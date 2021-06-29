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
}

func NewClient(logger *log.Logger, servers []string) *Client {
	client := &Client{
		Logger:  logger,
		servers: servers,
	}
	return client
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
		c.conn = c.connectUntilSuccess()
	}
	return c.conn
}

func (c *Client) getWatchConn() *zk.Conn {
	if c.watchConn == nil {
		c.watchConn = c.connectUntilSuccess()
	}
	return c.watchConn
}

func (c *Client) Exists(key string) (exist bool) {
	defer func() {
		c.Debugw("check zk exsit",
			"key", key,
			"exist", exist,
		)
	}()
	var err error
	exist, _, err = c.getConn().Exists(key)
	for err != nil {
		c.Errorw("zk check exists failed",
			"key", key,
			"error", err,
		)
		time.Sleep(time.Second)
	}
	return
}

func (c *Client) Create(key string) {
	flag := int32(0)
	acl := zk.WorldACL(zk.PermAll)
	_, err := c.getConn().Create(key, []byte("douyudouyu"), flag, acl)
	if err != nil {
		c.Errorw("zk create key failed",
			"error", err,
			"key", key,
		)
	}
}

func (c *Client) Get(key string) (value string, exist bool) {
	defer func() {
		c.Debugw("zk get",
			"key", key,
			"value", value,
			"exist", exist,
		)
	}()
	v, _, err := c.getConn().Get(key)
	for err != nil {
		if err == zk.ErrNoNode {
			return "", false
		}
		c.Errorw("zk get failed",
			"key", key,
			"error", err,
		)
		time.Sleep(time.Second)
	}
	value = string(v)
	exist = true
	return
}

// Delete 暂时不用
func (c *Client) Delete(key string) {
	c.Debugw("zk delete",
		"key", key,
	)
	_, s, err := c.getConn().Get(key)
	for err != nil {
		c.Errorw("zk delete failed",
			"key", key,
			"error", err,
		)
		time.Sleep(time.Second)
	}
	// TODO: 提升健壮性，处理删除冲突，进行重试
	err = c.getConn().Delete(key, s.Version)
	for err != nil {
		c.Errorw("zk delete failed",
			"key", key,
			"error", err,
		)
		time.Sleep(time.Second)
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

func (c *Client) connect() (conn *zk.Conn, err error) {
	c.Debugw("zk connect",
		"servers", c.servers,
	)
	conn, _, err = zk.Connect(c.servers, time.Second*10)
	return
}

func (c *Client) connectUntilSuccess() *zk.Conn {
	conn, err := c.connect()
	for err != nil { // 如果连接失败，一直重试，直到成功
		c.Errorw("failed to connect to zooKeeper, retrying...",
			"error", err,
			"servers", c.servers,
		)
		time.Sleep(time.Second)
		conn, err = c.connect()
	}
	return conn
}
