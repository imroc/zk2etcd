package etcd

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/imroc/zk2etcd/pkg/log"
	flag "github.com/spf13/pflag"
	"io/ioutil"
	"strings"
)

var client *Client

func Init(o *Options) {
	client = o.buildClient()
}

type Options struct {
	EtcdServers  string
	EtcdCaFile   string
	EtcdCertFile string
	EtcdKeyFile  string
}

func (o *Options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.EtcdServers, "etcd-servers", "", "comma-separated list of etcd servers address")
	fs.StringVar(&o.EtcdCaFile, "etcd-cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle")
	fs.StringVar(&o.EtcdCertFile, "etcd-cert", "", "identify secure client using this TLS certificate file")
	fs.StringVar(&o.EtcdKeyFile, "etcd-key", "", "identify secure client using this TLS key file")
}

func (o *Options) buildClient() *Client {
	var tlsConfig *tls.Config
	var etcdCert []tls.Certificate
	var rootCertPool *x509.CertPool

	if o.EtcdCertFile != "" && o.EtcdKeyFile != "" { // 加载 etcd client 证书
		etcdClientCert, err := tls.LoadX509KeyPair(o.EtcdCertFile, o.EtcdKeyFile)
		if err != nil {
			log.Panicw("load client tls error",
				"error", err,
			)
		}
		etcdCert = append(etcdCert, etcdClientCert)
	}

	if o.EtcdCaFile != "" { // 加载 etcd CA 证书
		etcdCA, err := ioutil.ReadFile(o.EtcdCaFile)
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
	return NewClient(strings.Split(o.EtcdServers, ","), tlsConfig)
}
