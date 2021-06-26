package main

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/imroc/zk2etcd/pkg/controller"
	"github.com/imroc/zk2etcd/pkg/etcd"
	"github.com/imroc/zk2etcd/pkg/log"
	"github.com/imroc/zk2etcd/pkg/zookeeper"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	zookeeperServers string
	zookeeperPrefix  string
	etcdServers      string
	logLevel         string
	etcdCaFile       string
	etcdCertFile     string
	etcdKeyFile      string
	concurrent       uint
)

var zkClient *zookeeper.Client
var etcdClient *etcd.Client
var logger *log.Logger

func initBase() {
	logger = log.New(logLevel)
	zkClient = zookeeper.NewClient(logger, strings.Split(zookeeperServers, ","))

	var tlsConfig *tls.Config
	var etcdCert []tls.Certificate
	var rootCertPool *x509.CertPool

	if etcdCertFile != "" && etcdKeyFile != "" { // 加载 etcd client 证书
		etcdClientCert, err := tls.LoadX509KeyPair(etcdCertFile, etcdKeyFile)
		if err != nil {
			logger.Panicw("load client tls error",
				"error", err,
			)
		}
		etcdCert = append(etcdCert, etcdClientCert)
	}

	if etcdCaFile != "" { // 加载 etcd CA 证书
		etcdCA, err := ioutil.ReadFile(etcdCaFile)
		if err != nil {
			logger.Panicw("read etcd ca file error",
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
	etcdClient = etcd.NewClient(logger, strings.Split(etcdServers, ","), tlsConfig)
}

// GetRootCmd returns the root of the cobra command-tree.
func GetRootCmd(args []string) *cobra.Command {
	stopChan := make(chan struct{}, 1)
	rootCmd := &cobra.Command{
		Use:   "zk2etcd",
		Short: "zookeeper sync to etcd",

		DisableAutoGenTag: true,
		Long:              `sync data from zookeeper to etcd`,
		Run: func(cmd *cobra.Command, args []string) {
			initBase()

			c := controller.New(zkClient, zookeeperPrefix, etcdClient, logger, concurrent)
			go c.Run(stopChan)

			// TODO: 实现真正优雅停止
			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
			<-signalChan
			stopChan <- struct{}{}
		},
	}

	rootCmd.SetArgs(args)
	rootCmd.PersistentFlags().StringVar(&zookeeperServers, "zookeeper-servers", "", "comma-separated list of zookeeper servers address")
	rootCmd.PersistentFlags().StringVar(&zookeeperPrefix, "zookeeper-prefix", "/dubbo", "comma-separated list of zookeeper path prefix to be synced")
	rootCmd.PersistentFlags().StringVar(&etcdServers, "etcd-servers", "", "comma-separated list of etcd servers address")
	rootCmd.PersistentFlags().StringVar(&etcdCaFile, "etcd-cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle")
	rootCmd.PersistentFlags().StringVar(&etcdCertFile, "etcd-cert", "", "identify secure client using this TLS certificate file")
	rootCmd.PersistentFlags().StringVar(&etcdKeyFile, "etcd-key", "", "identify secure client using this TLS key file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log output level，possible values: 'debug', 'info', 'warn', 'error', 'panic', 'fatal'")
	rootCmd.PersistentFlags().UintVar(&concurrent, "concurrent", 20, "the concurreny of syncing worker")

	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}
