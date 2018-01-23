package logger

import (
	"github.com/Sirupsen/logrus"
	"github.com/hillrnate/pritunl-zero/constants"
	"github.com/hillrnate/pritunl-zero/requires"
	"os"
	"strings"
)

var (
	buffer  = make(chan *logrus.Entry, 32)
	senders = []sender{}
)

func initSender() {
	for _, sndr := range senders {
		sndr.Init()
	}

	go func() {
		for {
			entry := <-buffer

			if constants.Interrupt {
				return
			}

			if strings.HasPrefix(entry.Message, "logger:") {
				continue
			}

			for _, sndr := range senders {
				sndr.Parse(entry)
			}
		}
	}()
}

func Init() {
	logrus.SetFormatter(&formatter{})
	logrus.AddHook(&logHook{})
	logrus.SetOutput(os.Stderr)
	logrus.SetLevel(logrus.InfoLevel)
}

func init() {
	module := requires.New("logger")
	module.After("config")

	module.Handler = func() (err error) {
		initSender()
		return
	}
}
