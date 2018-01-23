package main

import (
	"flag"
	"fmt"
	"github.com/hillrnate/pritunl-zero/cmd"
	"github.com/hillrnate/pritunl-zero/constants"
	"github.com/hillrnate/pritunl-zero/logger"
	"github.com/hillrnate/pritunl-zero/requires"
	"github.com/hillrnate/pritunl-zero/task"
	"time"
)

const help = `
Usage: pritunl-zero COMMAND

Commands:
  version     Show version
  mongo       Set MongoDB URI
  set         Set a setting
  unset       Unset a setting
  start       Start node
  clear-logs  Clear logs
  export-ssh  Export SSH authorities for emergency client
`

func Init() {
	logger.Init()
	requires.Init()
	task.Init()
}

func main() {
	defer time.Sleep(1 * time.Second)

	flag.Parse()

	switch flag.Arg(0) {
	case "start":
		if flag.Arg(1) == "--debug" {
			constants.Production = false
		}

		Init()
		err := cmd.Node()
		if err != nil {
			panic(err)
		}
		return
	case "version":
		fmt.Printf("pritunl-zero v%s\n", constants.Version)
		return
	case "mongo":
		logger.Init()
		err := cmd.Mongo()
		if err != nil {
			panic(err)
		}
		return
	case "reset-id":
		logger.Init()
		err := cmd.ResetId()
		if err != nil {
			panic(err)
		}
		return
	case "set":
		Init()
		err := cmd.SettingsSet()
		if err != nil {
			panic(err)
		}
		return
	case "unset":
		Init()
		err := cmd.SettingsUnset()
		if err != nil {
			panic(err)
		}
		return
	case "export-ssh":
		Init()
		err := cmd.ExportSsh()
		if err != nil {
			panic(err)
		}
		return
	case "clear-logs":
		Init()
		err := cmd.ClearLogs()
		if err != nil {
			panic(err)
		}
		return
	}

	fmt.Println(help)
}
