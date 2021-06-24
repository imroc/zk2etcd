package controller

import (
	"context"
	"fmt"
	"github.com/go-zookeeper/zk"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"log"
	"path/filepath"
	"strings"
	"time"
)

type Controller struct {
	zkAddr     string
	zkPrefix   string
	etcdAddr   string
	zkConn     *zk.Conn
	zkConn2     *zk.Conn
	etcdClient *clientv3.Client
}

func New(zkAddr, zkPrefix, etcdAddr string) *Controller {
	return &Controller{
		zkAddr:   zkAddr,
		zkPrefix: zkPrefix,
		etcdAddr: etcdAddr,
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
			log.Printf("failed to check path existence : %v\n", err)
		}
		if !exists {
			log.Println("zookeeper path " + c.zkPrefix + " doesn't exist, wait until it's created")
		}
		time.Sleep(time.Second * 2)
	}

	// 启动全量同步一次
	c.syncAll(c.zkPrefix)

	// 继续同步增量
	w := zookeeper.New(conn, c.zkPrefix)
	w.Run(stop)
}

func (c *Controller) syncAll(path string) {
	fmt.Println("syncing all", path)
	c.syncPath(path)
	children, _, err := c.zkConn2.Children(path)
	if err != nil {
		fmt.Println("ERROR:", err)
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
		log.Printf("failed to connect to ZooKeeper %s ,retrying : %v\n", hosts, err)
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
		fmt.Printf("ignore event: %+v\n", event)
	}
}

func (c *Controller) syncPath(path string) {
	v, _, err := c.zkConn2.Get(path)
	if err != nil {
		fmt.Println(err)
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
	fmt.Println("sync children", path)
	children, _, err := c.zkConn2.Children(path)
	fmt.Println("got children", children)
	if err != nil {
		fmt.Println("ERROR:", err)
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
		fmt.Println(err)
	}
}

func (c *Controller) etcdGet(path string) (string, bool) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	resp, err := c.etcdClient.Get(ctx, path)
	if err != nil {
		fmt.Println(err)
		return "", false
	}
	if len(resp.Kvs) != 0 {
		return string(resp.Kvs[0].Value), true
	}
	return "", false
}
