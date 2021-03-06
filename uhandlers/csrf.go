package uhandlers

import (
	"github.com/gin-gonic/gin"
	"github.com/hillrnate/pritunl-zero/authorizer"
	"github.com/hillrnate/pritunl-zero/csrf"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/utils"
)

type csrfData struct {
	Token string `json:"token"`
	Theme string `json:"theme"`
}

func csrfGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	authr := c.MustGet("authorizer").(*authorizer.Authorizer)

	usr, err := authr.GetUser(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	token, err := csrf.NewToken(db, authr.SessionId())
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	data := &csrfData{
		Token: token,
		Theme: usr.Theme,
	}
	c.JSON(200, data)
}
