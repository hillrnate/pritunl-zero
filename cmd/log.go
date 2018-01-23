package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/log"
)

func ClearLogs() (err error) {
	db := database.GetDatabase()
	defer db.Close()

	err = log.Clear(db)
	if err != nil {
		return
	}

	logrus.Info("cmd.log: Logs cleared")

	return
}
