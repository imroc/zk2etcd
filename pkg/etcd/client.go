package etcd

import (
	"context"
	"crypto/tls"
	"github.com/imroc/zk2etcd/pkg/log"
	clientv3 "go.etcd.io/etcd/client/v3"
	"time"
)

var defaultTimeout = 5 * time.Second

func timeoutContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), defaultTimeout)
	return ctx
}

type Client struct {
	*log.Logger
	servers []string
	*clientv3.Client
	tls *tls.Config
}

func NewClient(logger *log.Logger, servers []string, tls *tls.Config) *Client {
	client := &Client{
		Logger:  logger,
		servers: servers,
		tls:     tls,
	}
	err := client.init()
	if err != nil {
		logger.Errorw("init etcd client failed",
			"error", err,
		)
	}
	return client
}

func (c *Client) init() error {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   c.servers,
		DialTimeout: 5 * time.Second,
		Logger:      c.Desugar(),
		TLS:         c.tls,
	})
	if err != nil {
		return err
	}
	c.Client = client
	return nil
}

func (c *Client) Put(key, value string) {
	c.Debugw("etcd put",
		"key", key,
		"value", value,
	)
	_, err := c.Client.Put(timeoutContext(), key, value)
	for err != nil {
		c.Errorw("etcd failed to put",
			"key", key,
			"value", value,
			"error", err,
		)
		time.Sleep(time.Second)
		_, err = c.Client.Put(timeoutContext(), key, value)
	}
}

func (c *Client) Delete(key string) {
	c.Debugw("etcd delete key",
		"key", key,
	)
	_, err := c.Client.Delete(timeoutContext(), key, clientv3.WithPrefix())
	for err != nil {
		c.Errorw("etcd failed to delete",
			"key", key,
			"error", err,
		)
		time.Sleep(time.Second)
		_, err = c.Client.Delete(timeoutContext(), key, clientv3.WithPrefix())
	}
}

func (c *Client) Get(key string) (value string, ok bool) {
	defer func() {
		c.Debugw("etcd get",
			"key", key,
			"value", value,
			"exist", ok,
		)
	}()

	resp, err := c.Client.Get(timeoutContext(), key)

	for err != nil {
		c.Errorw("etcd failed to get",
			"key", key,
			"error", err,
		)
		time.Sleep(time.Second)
		resp, err = c.Client.Get(timeoutContext(), key)
	}
	if len(resp.Kvs) != 0 {
		value = string(resp.Kvs[0].Value)
		ok = true
	}
	return
}

func (c *Client) ListAllKeys(key string) []string {
	resp, err := c.Client.Get(timeoutContext(), key, clientv3.WithPrefix())
	for err != nil {
		c.Errorw("etcd failed to get",
			"key", key,
			"error", err,
		)
		time.Sleep(time.Second)
		resp, err = c.Client.Get(timeoutContext(), key)
	}
	keys := []string{}
	for _, kvs := range resp.Kvs {
		keys = append(keys, string(kvs.Key))
	}
	c.Debugw("etcd list all",
		"prefix", key,
	)
	return keys
}
