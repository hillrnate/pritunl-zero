package mhandlers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/demo"
	"github.com/hillrnate/pritunl-zero/log"
	"github.com/hillrnate/pritunl-zero/utils"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"strings"
)

type logsData struct {
	Logs  []*log.Entry `json:"logs"`
	Count int          `json:"count"`
}

func logGet(c *gin.Context) {
	if demo.IsDemo() {
		c.JSON(200, demo.Logs[1])
		return
	}

	db := c.MustGet("db").(*database.Database)

	logId, ok := utils.ParseObjectId(c.Param("log_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	usr, err := log.Get(db, logId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	c.JSON(200, usr)
}

func logsGet(c *gin.Context) {
	if demo.IsDemo() {
		data := &logsData{
			Logs:  demo.Logs,
			Count: len(demo.Logs),
		}

		c.JSON(200, data)
		return
	}

	db := c.MustGet("db").(*database.Database)

	pageStr := c.Query("page")
	page, _ := strconv.Atoi(pageStr)
	pageCountStr := c.Query("page_count")
	pageCount, _ := strconv.Atoi(pageCountStr)

	query := bson.M{}

	message := strings.TrimSpace(c.Query("message"))
	if message != "" {
		query["message"] = &bson.M{
			"$regex":   fmt.Sprintf(".*%s.*", message),
			"$options": "i",
		}
	}

	level := strings.TrimSpace(c.Query("level"))
	if level != "" {
		query["level"] = level
	}

	logs, count, err := log.GetAll(db, &query, page, pageCount)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	data := &logsData{
		Logs:  logs,
		Count: count,
	}

	c.JSON(200, data)
}
