package mhandlers

import (
	"fmt"
	"github.com/dropbox/godropbox/container/set"
	"github.com/gin-gonic/gin"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/demo"
	"github.com/hillrnate/pritunl-zero/event"
	"github.com/hillrnate/pritunl-zero/user"
	"github.com/hillrnate/pritunl-zero/utils"
	"gopkg.in/mgo.v2/bson"
	"strconv"
	"strings"
	"time"
)

type userData struct {
	Id             bson.ObjectId `json:"id"`
	Type           string        `json:"type"`
	Username       string        `json:"username"`
	Password       string        `json:"password"`
	Keybase        string        `json:"keybase"`
	Roles          []string      `json:"roles"`
	Administrator  string        `json:"administrator"`
	Permissions    []string      `json:"permissions"`
	GenerateSecret bool          `json:"generate_secret"`
	Disabled       bool          `json:"disabled"`
	ActiveUntil    time.Time     `json:"active_until"`
}

type usersData struct {
	Users []*user.User `json:"users"`
	Count int          `json:"count"`
}

func userGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)

	userId, ok := utils.ParseObjectId(c.Param("user_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	usr, err := user.Get(db, userId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if demo.IsDemo() {
		if usr.Username == "demo" {
			usr.LastActive = time.Now()
		} else {
			usr.LastActive = time.Time{}
		}
	}

	usr.Secret = ""

	c.JSON(200, usr)
}

func userPut(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &userData{}

	userId, ok := utils.ParseObjectId(c.Param("user_id"))
	if !ok {
		utils.AbortWithStatus(c, 400)
		return
	}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	usr, err := user.Get(db, userId)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	showSecret := false
	if usr.Type != data.Type {
		if data.Type == user.Api {
			usr.GenerateToken()
			showSecret = true
		} else {
			usr.Token = ""
			usr.Secret = ""
		}
	}

	usr.Type = data.Type
	usr.Username = data.Username
	usr.Keybase = data.Keybase
	usr.Roles = data.Roles
	usr.Administrator = data.Administrator
	usr.Permissions = data.Permissions
	usr.Disabled = data.Disabled
	usr.ActiveUntil = data.ActiveUntil

	if usr.Disabled {
		usr.ActiveUntil = time.Time{}
	}

	if usr.Type == user.Api && data.GenerateSecret {
		usr.GenerateToken()
		showSecret = true
	}

	fields := set.NewSet(
		"type",
		"token",
		"secret",
		"username",
		"keybase",
		"roles",
		"administrator",
		"permissions",
		"disabled",
		"active_until",
	)

	if usr.Type == user.Local && data.Password != "" {
		err = usr.SetPassword(data.Password)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		fields.Add("password")
	} else if usr.Type != user.Local && usr.Password != "" {
		usr.Password = ""
		fields.Add("password")
	}

	errData, err := usr.Validate(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	err = usr.CommitFields(db, fields)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.PublishDispatch(db, "user.change")

	if !showSecret {
		usr.Secret = ""
	}

	c.JSON(200, usr)
}

func userPost(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &userData{}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	usr := &user.User{
		Type:          data.Type,
		Username:      data.Username,
		Keybase:       data.Keybase,
		Roles:         data.Roles,
		Administrator: data.Administrator,
		Permissions:   data.Permissions,
		Disabled:      data.Disabled,
		ActiveUntil:   data.ActiveUntil,
	}

	if usr.Disabled {
		usr.ActiveUntil = time.Time{}
	}

	if usr.Type == user.Local && data.Password != "" {
		err = usr.SetPassword(data.Password)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}
	}

	if usr.Type == user.Api {
		usr.GenerateToken()
	}

	errData, err := usr.Validate(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	err = usr.Insert(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.PublishDispatch(db, "user.change")

	c.JSON(200, usr)
}

func usersGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)

	page, _ := strconv.Atoi(c.Query("page"))
	pageCount, _ := strconv.Atoi(c.Query("page_count"))

	query := bson.M{}

	username := strings.TrimSpace(c.Query("username"))
	if username != "" {
		query["username"] = &bson.M{
			"$regex":   fmt.Sprintf(".*%s.*", username),
			"$options": "i",
		}
	}

	keybase := strings.TrimSpace(c.Query("keybase"))
	if keybase != "" {
		query["keybase"] = &bson.M{
			"$regex":   fmt.Sprintf(".*%s.*", keybase),
			"$options": "i",
		}
	}

	role := strings.TrimSpace(c.Query("role"))
	if role != "" {
		query["roles"] = role
	}

	typ := strings.TrimSpace(c.Query("type"))
	if typ != "" {
		query["type"] = typ
	}

	administrator := c.Query("administrator")
	switch administrator {
	case "true":
		query["administrator"] = "super"
		break
	case "false":
		query["administrator"] = ""
		break
	}

	disabled := c.Query("disabled")
	switch disabled {
	case "true":
		query["disabled"] = true
		break
	case "false":
		query["disabled"] = false
		break
	}

	users, count, err := user.GetAll(db, &query, page, pageCount)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if demo.IsDemo() {
		for _, usr := range users {
			if usr.Username == "demo" {
				usr.LastActive = time.Now()
			} else {
				usr.LastActive = time.Time{}
			}
		}
	}

	for _, usr := range users {
		usr.Secret = ""
	}

	data := &usersData{
		Users: users,
		Count: count,
	}

	c.JSON(200, data)
}

func usersDelete(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := []bson.ObjectId{}

	err := c.Bind(&data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	errData, err := user.Remove(db, data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	event.PublishDispatch(db, "user.change")

	c.JSON(200, nil)
}
