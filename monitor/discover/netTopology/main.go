package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"netTopology/discover"
	. "netTopology/internal"
)

func init() {
	// add your snmp communities, default is "public" or "publicv2"
	communities := map[string]string{}
	for k, v := range communities {
		CommunityMap.SetDefault(k, v)
	}
}

func main() {
	logrus.Info("start...")
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	md := discover.MasterDiscover("master")
	discover.NetworkMonitorInit(term, &md)
	<-term
	logrus.Info("Received SIGTERM, exiting ...")
}
