package validator

import (
	"github.com/dropbox/godropbox/container/set"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/errortypes"
	"github.com/hillrnate/pritunl-zero/policy"
	"github.com/hillrnate/pritunl-zero/service"
	"github.com/hillrnate/pritunl-zero/user"
	"gopkg.in/mgo.v2/bson"
	"net/http"
)

func ValidateAdmin(db *database.Database, usr *user.User,
	isApi bool, r *http.Request) (secProvider bson.ObjectId,
	errData *errortypes.ErrorData, err error) {

	if usr.Disabled || usr.Administrator != "super" {
		errData = &errortypes.ErrorData{
			Error:   "unauthorized",
			Message: "Not authorized",
		}
		return
	}

	if !isApi {
		policies, e := policy.GetRoles(db, usr.Roles)
		if e != nil {
			err = e
			return
		}

		for _, polcy := range policies {
			if polcy.AdminSecondary != "" {
				secProvider = polcy.AdminSecondary
				break
			}
		}
	}

	return
}

func ValidateUser(db *database.Database, usr *user.User,
	isApi bool, r *http.Request) (secProvider bson.ObjectId,
	errData *errortypes.ErrorData, err error) {

	if usr.Disabled {
		errData = &errortypes.ErrorData{
			Error:   "unauthorized",
			Message: "Not authorized",
		}
		return
	}

	if !isApi {
		policies, e := policy.GetRoles(db, usr.Roles)
		if e != nil {
			err = e
			return
		}

		for _, polcy := range policies {
			errData, err = polcy.ValidateUser(db, usr, r)
			if err != nil || errData != nil {
				return
			}
		}

		for _, polcy := range policies {
			if polcy.UserSecondary != "" {
				secProvider = polcy.UserSecondary
				break
			}
		}
	}

	return
}

func ValidateProxy(db *database.Database, usr *user.User,
	isApi bool, srvc *service.Service, r *http.Request) (
	secProvider bson.ObjectId, errData *errortypes.ErrorData, err error) {

	if usr.Disabled {
		errData = &errortypes.ErrorData{
			Error:   "unauthorized",
			Message: "Not authorized",
		}
		return
	}

	usrRoles := set.NewSet()
	for _, role := range usr.Roles {
		usrRoles.Add(role)
	}

	roleMatch := false
	for _, role := range srvc.Roles {
		if usrRoles.Contains(role) {
			roleMatch = true
			break
		}
	}

	if !roleMatch {
		errData = &errortypes.ErrorData{
			Error:   "service_unauthorized",
			Message: "Not authorized for service",
		}
		return
	}

	if !isApi {
		policies, e := policy.GetService(db, srvc.Id)
		if e != nil {
			err = e
			return
		}

		for _, polcy := range policies {
			errData, err = polcy.ValidateUser(db, usr, r)
			if err != nil || errData != nil {
				return
			}
		}

		for _, polcy := range policies {
			if polcy.ProxySecondary != "" {
				secProvider = polcy.ProxySecondary
				break
			}
		}

		policies, err = policy.GetRoles(db, usr.Roles)
		if err != nil {
			return
		}

		for _, polcy := range policies {
			errData, err = polcy.ValidateUser(db, usr, r)
			if err != nil || errData != nil {
				return
			}
		}

		for _, polcy := range policies {
			if polcy.ProxySecondary != "" {
				secProvider = polcy.ProxySecondary
				break
			}
		}
	}

	return
}
