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
	_, err := c.Client.Put(timeoutContext(), key, value)
	if err != nil {
		c.Errorw("etcd failed to put",
			"key", key,
			"value", value,
			"error", err,
		)
	}
}

func (c *Client) Delete(key string) {
	c.Infow("etcd delete key",
		"key", key,
	)
	_, err := c.Client.Delete(timeoutContext(), key)
	if err != nil {
		c.Errorw("etcd failed to delete",
			"key", key,
			"error", err,
		)
	}
}

func (c *Client) Get(key string) (string, bool) {
	resp, err := c.Client.Get(timeoutContext(), key)
	for err != nil {
		c.Errorw("etcd failed to get",
			"key", key,
			"error", err,
		)
		time.Sleep(time.Second)
	}
	if len(resp.Kvs) != 0 {
		return string(resp.Kvs[0].Value), true
	}
	return "", false
}
