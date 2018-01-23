package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/hillrnate/pritunl-zero/config"
	"github.com/hillrnate/pritunl-zero/constants"
	"github.com/hillrnate/pritunl-zero/node"
	"github.com/hillrnate/pritunl-zero/router"
	"gopkg.in/mgo.v2/bson"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Node() (err error) {
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	nde := &node.Node{
		Id:   bson.ObjectIdHex(config.Config.NodeId),
		Type: node.Management,
	}
	err = nde.Init()
	if err != nil {
		return
	}

	routr := &router.Router{}

	routr.Init()

	go func() {
		err = routr.Run()
		if err != nil {
			panic(err)
		}
	}()

	<-sig
	logrus.Info("cmd.node: Shutting down")
	go routr.Shutdown()
	if constants.Production {
		time.Sleep(10 * time.Second)
	} else {
		time.Sleep(300 * time.Millisecond)
	}

	return
}
