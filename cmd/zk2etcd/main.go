package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
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

// GetRootCmd returns the root of the cobra command-tree.
func GetRootCmd(args []string) *cobra.Command {
	var zkAddr, zkPrefix, etcdAddr, logLevel, etcdCaFile, etcdCertFile, etcdKeyFile string
	stopChan := make(chan struct{}, 1)
	rootCmd := &cobra.Command{
		Use:   "zk2etcd",
		Short: "zookeeper sync to etcd",

		DisableAutoGenTag: true,
		Long:              `sync data from zookeeper to etcd`,
		Run: func(cmd *cobra.Command, args []string) {
			logger := log.New(logLevel)
			zkServers := strings.Split(zkAddr, ",")
			zkClient := zookeeper.NewClient(logger, zkServers)
			etcdServers := strings.Split(etcdAddr, ",")

			var tlsConfig *tls.Config
			var etcdCert []tls.Certificate
			var rootCertPool *x509.CertPool
			if etcdCertFile != "" && etcdKeyFile != "" {
				etcdClientCert, err := tls.LoadX509KeyPair(etcdCertFile, etcdKeyFile)
				if err != nil {
					logger.Panicw("load client tls error",
						"error", err,
					)
				}
				etcdCert = append(etcdCert, etcdClientCert)
			}

			if etcdCaFile != "" {
				// 为了保证 HTTPS 链接可信，需要预先加载目标证书签发机构的 CA 根证书
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
				etcdClient := etcd.NewClient(logger, etcdServers, tlsConfig)
				c := controller.New(zkClient, zkPrefix, etcdClient, logger)
				go c.Run(stopChan)
				signalChan := make(chan os.Signal, 1)
				signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
				<-signalChan
				stopChan <- struct{}{}
			}
		},
	}

	rootCmd.SetArgs(args)
	rootCmd.PersistentFlags().StringVar(&zkAddr, "zookeeper-servers", "", "comma-separated list of zookeeper servers address")
	rootCmd.PersistentFlags().StringVar(&zkPrefix, "zookeeper-prefix", "/dubbo", "the zookeeper path prefix to be synced")
	rootCmd.PersistentFlags().StringVar(&etcdAddr, "etcd-servers", "", "comma-separated list of etcd servers address")
	rootCmd.PersistentFlags().StringVar(&etcdCaFile, "etcd-cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle")
	rootCmd.PersistentFlags().StringVar(&etcdCertFile, "etcd-cert", "", "identify secure client using this TLS certificate file")
	rootCmd.PersistentFlags().StringVar(&etcdKeyFile, "etcd-key", "", "identify secure client using this TLS key file")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log output level，possible values: 'debug', 'info', 'warn', 'error', 'panic', 'fatal'")

	rootCmd.AddCommand(newVersionCmd())

	return rootCmd
}

func main() {
	rootCmd := GetRootCmd(os.Args[1:])
	if err := rootCmd.Execute(); err != nil {
		//exitCode := cmd.GetExitCode(err)
		//os.Exit(exitCode)
		fmt.Println(err)
	}
}
