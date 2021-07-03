package main

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	flag "github.com/spf13/pflag"
	"io/ioutil"
	"strings"
)

type Common struct {
	zookeeperServers       string
	zookeeperPrefix        string
	zookeeperExcludePrefix string // TODO: 先简单实现 exclude，后续优化 exlude 判断的性能
	etcdServers            string
	logLevel               string
	etcdCaFile             string
	etcdCertFile           string
	etcdKeyFile            string
}

func (c *Common) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.zookeeperServers, "zookeeper-servers", "", "comma-separated list of zookeeper servers address")
	fs.StringVar(&c.zookeeperPrefix, "zookeeper-prefix", "/dubbo", "comma-separated list of zookeeper path prefix to be synced")
	fs.StringVar(&c.zookeeperExcludePrefix, "zookeeper-exclude-prefix", "/dubbo/config", "comma-separated list of zookeeper path prefix to be excluded")
	fs.StringVar(&c.etcdServers, "etcd-servers", "", "comma-separated list of etcd servers address")
	fs.StringVar(&c.etcdCaFile, "etcd-cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle")
	fs.StringVar(&c.etcdCertFile, "etcd-cert", "", "identify secure client using this TLS certificate file")
	fs.StringVar(&c.etcdKeyFile, "etcd-key", "", "identify secure client using this TLS key file")
	fs.StringVar(&c.logLevel, "log-level", "info", "log output level，possible values: 'debug', 'info', 'warn', 'error', 'panic', 'fatal'")
}

func (c *Common) GetAll() (etcdClient *etcd.Client, zkPrefixes, zkExcludePrefixes []string) {
	zkPrefixes = strings.Split(c.zookeeperPrefix, ",")
	if len(zkPrefixes) == 0 {
		zkPrefixes = append(zkPrefixes, "/")
	}
	zkExcludePrefixes = strings.Split(c.zookeeperExcludePrefix, ",")

	var tlsConfig *tls.Config
	var etcdCert []tls.Certificate
	var rootCertPool *x509.CertPool

	if c.etcdCertFile != "" && c.etcdKeyFile != "" { // 加载 etcd client 证书
		etcdClientCert, err := tls.LoadX509KeyPair(c.etcdCertFile, c.etcdKeyFile)
		if err != nil {
			log.Panicw("load client tls error",
				"error", err,
			)
		}
		etcdCert = append(etcdCert, etcdClientCert)
	}

	if c.etcdCaFile != "" { // 加载 etcd CA 证书
		etcdCA, err := ioutil.ReadFile(c.etcdCaFile)
		if err != nil {
			log.Panicw("read etcd ca file error",
				"error", err,
			)
		}
		rootCertPool = x509.NewCertPool()
		rootCertPool.AppendCertsFromPEM(etcdCA)
	}

	if len(etcdCert) > 0 || rootCertPool != nil {
		tlsConfig = &tls.Config{
			RootCAs:      rootCertPool,
			Certificates: etcdCert,
		}
	}
	etcdClient = etcd.NewClient(strings.Split(c.etcdServers, ","), tlsConfig)
	return
}
