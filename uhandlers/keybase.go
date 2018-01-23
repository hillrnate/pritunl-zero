package uhandlers

import (
	"github.com/gin-gonic/gin"
	"github.com/hillrnate/pritunl-zero/audit"
	"github.com/hillrnate/pritunl-zero/authorizer"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/demo"
	"github.com/hillrnate/pritunl-zero/errortypes"
	"github.com/hillrnate/pritunl-zero/event"
	"github.com/hillrnate/pritunl-zero/keybase"
	"github.com/hillrnate/pritunl-zero/secondary"
	"github.com/hillrnate/pritunl-zero/ssh"
	"github.com/hillrnate/pritunl-zero/user"
	"github.com/hillrnate/pritunl-zero/utils"
	"time"
)

type keybaseAssociateData struct {
	Username string `json:"username"`
}

type keybaseValidateData struct {
	Token     string `json:"token"`
	Message   string `json:"message,omitempty"`
	Signature string `json:"signature,omitempty"`
}

func keybaseInfoGet(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	token := c.Param("token")

	asc, err := keybase.GetAssociation(db, token)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	info, err := asc.GetInfo()
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	c.JSON(200, info)

	return
}

func keybaseValidatePut(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	authr := c.MustGet("authorizer").(*authorizer.Authorizer)
	data := &keybaseValidateData{}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	usr, err := authr.GetUser(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	asc, err := keybase.GetAssociation(db, data.Token)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	err, errData := asc.Validate(data.Signature)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(406, errData)
		return
	}

	err = audit.New(
		db,
		c.Request,
		usr.Id,
		audit.KeybaseAssociationApprove,
		audit.Fields{
			"keybase_username": asc.Username,
		},
	)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	err, errData = asc.Approve(db, usr)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.Publish(db, "keybase_association", asc.Id)

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	c.Status(200)
}

func keybaseValidateDelete(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	authr := c.MustGet("authorizer").(*authorizer.Authorizer)
	data := &keybaseValidateData{}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	usr, err := authr.GetUser(db)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	asc, err := keybase.GetAssociation(db, data.Token)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	err, errData := asc.Validate(data.Signature)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	err = audit.New(
		db,
		c.Request,
		usr.Id,
		audit.KeybaseAssociationDeny,
		audit.Fields{
			"keybase_username": asc.Username,
		},
	)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	err = asc.Deny(db, usr)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	event.Publish(db, "keybase_association", asc.Id)

	c.Status(200)
}

func keybaseCheckPut(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &keybaseValidateData{}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	asc, err := keybase.GetAssociation(db, data.Token)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	err, errData := asc.Validate(data.Signature)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(400, errData)
		return
	}

	_, err = user.GetKeybase(db, asc.Username)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	c.Status(200)
}

func keybaseAssociatePost(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &keybaseAssociateData{}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	asc, err := keybase.NewAssociation(db, data.Username)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	resp := &keybaseValidateData{
		Token:   asc.Id,
		Message: asc.Message(),
	}

	c.JSON(200, resp)
}

func keybaseAssociateGet(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	token := c.Param("token")

	asc, err := keybase.GetAssociation(db, token)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}
	token = asc.Id

	sync := func() {
		asc, err = keybase.GetAssociation(db, token)
		if err != nil {
			switch err.(type) {
			case *database.NotFoundError:
				utils.AbortWithStatus(c, 404)
				break
			default:
				utils.AbortWithError(c, 500, err)
			}
			return
		}
	}

	update := func() bool {
		switch asc.State {
		case ssh.Approved:
			c.Status(200)
			return true
		case ssh.Denied:
			c.Status(401)
			return true
		}

		return false
	}

	if update() {
		return
	}

	start := time.Now()
	ticker := time.NewTicker(3 * time.Second)
	notify := make(chan bool, 3)

	listenerId := keybase.Register(token, func() {
		defer func() {
			recover()
		}()
		notify <- true
	})
	defer keybase.Unregister(token, listenerId)

	for {
		select {
		case <-ticker.C:
			if time.Since(start) > 29*time.Second {
				c.Status(205)
				return
			}

			sync()
			if update() {
				return
			}
		case <-notify:
			sync()
			if update() {
				return
			}
		}
	}
}

type keybaseChallengeData struct {
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
}

type keybaseChallengeRespData struct {
	Token             string `json:"token"`
	Message           string `json:"message"`
	Signature         string `json:"signature,omitempty"`
	SecondaryToken    string `json:"secondary_token"`
	SecondaryFactor   string `json:"secondary_factor"`
	SecondaryPasscode string `json:"secondary_passcode"`
}

type keybaseCertificateData struct {
	Token                  string      `json:"token"`
	Certificates           []string    `json:"certificates"`
	CertificateAuthorities []string    `json:"certificate_authorities"`
	Hosts                  []*ssh.Host `json:"hosts"`
}

func keybaseChallengePost(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &keybaseChallengeData{}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	chal, err := keybase.NewChallenge(db, data.Username, data.PublicKey)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	resp := &keybaseValidateData{
		Token:   chal.Id,
		Message: chal.Message(),
	}

	c.JSON(200, resp)
}

func keybaseChallengePut(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &keybaseChallengeRespData{}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	chal, err := keybase.GetChallenge(db, data.Token)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	secProviderId, errData, err := chal.Validate(
		db, c.Request, data.Signature)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	if errData != nil {
		c.JSON(406, errData)
		return
	}

	if secProviderId != "" {
		usr, err := chal.GetUser(db)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		secd, err := secondary.NewChallenge(
			db, usr.Id, secondary.Keybase, chal.Id, secProviderId)
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		data, err := secd.GetData()
		if err != nil {
			utils.AbortWithError(c, 500, err)
			return
		}

		c.JSON(201, data)
		return
	}

	cert, errData, err := chal.NewCertificate(db, c.Request)
	if err != nil {
		return
	}

	resp := &keybaseCertificateData{
		Token:                  chal.Id,
		Hosts:                  cert.Hosts,
		Certificates:           cert.Certificates,
		CertificateAuthorities: cert.CertificateAuthorities,
	}

	c.JSON(200, resp)
}

type keybaseSecondaryData struct {
	Token    string `json:"token"`
	Factor   string `json:"factor"`
	Passcode string `json:"passcode"`
}

func keybaseSecondaryPut(c *gin.Context) {
	if demo.Blocked(c) {
		return
	}

	db := c.MustGet("db").(*database.Database)
	data := &keybaseSecondaryData{}

	err := c.Bind(data)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	secd, err := secondary.Get(db, data.Token, secondary.Keybase)
	if err != nil {
		if _, ok := err.(*database.NotFoundError); ok {
			errData := &errortypes.ErrorData{
				Error:   "secondary_expired",
				Message: "Two-factor authentication has expired",
			}
			c.JSON(401, errData)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	errData, err := secd.Handle(db, c.Request, data.Factor, data.Passcode)
	if err != nil {
		if _, ok := err.(*secondary.IncompleteError); ok {
			c.Status(201)
		} else {
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	if errData != nil {
		c.JSON(401, errData)
		return
	}

	chal, err := keybase.GetChallenge(db, secd.ChallengeId)
	if err != nil {
		switch err.(type) {
		case *database.NotFoundError:
			utils.AbortWithStatus(c, 404)
			break
		default:
			utils.AbortWithError(c, 500, err)
		}
		return
	}

	cert, errData, err := chal.NewCertificate(db, c.Request)
	if err != nil {
		utils.AbortWithError(c, 500, err)
		return
	}

	resp := &keybaseCertificateData{
		Token:                  chal.Id,
		Hosts:                  cert.Hosts,
		Certificates:           cert.Certificates,
		CertificateAuthorities: cert.CertificateAuthorities,
	}

	c.JSON(200, resp)
}
