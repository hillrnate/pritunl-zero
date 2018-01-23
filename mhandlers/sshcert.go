package mhandlers

import (
	"github.com/gin-gonic/gin"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/demo"
	"github.com/hillrnate/pritunl-zero/ssh"
	"github.com/hillrnate/pritunl-zero/utils"
	"strconv"
)

type sshcertsData struct {
	Certificates []*ssh.Certificate `json:"certificates"`
	Count        int                `json:"count"`
}

func sshcertsGet(c *gin.Context) {
	if demo.IsDemo() {
		data := &sshcertsData{
			Certificates: demo.Sshcerts,
			Count:        len(demo.Sshcerts),
		}

		c.JSON(200, data)
		return
	}

	db := c.MustGet("db").(*database.Database)

	page, _ := strconv.Atoi(c.Query("page"))
	pageCount, _ := strconv.Atoi(c.Query("page_count"))

	userId, ok := utils.ParseObjectId(c.Param("user_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	certs, count, err := ssh.GetCertificates(db, userId, page, pageCount)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	data := &sshcertsData{
		Certificates: certs,
		Count:        count,
	}

	c.JSON(200, data)
}
