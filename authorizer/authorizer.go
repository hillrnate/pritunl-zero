package authorizer

import (
	"github.com/hillrnate/pritunl-zero/cookie"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/service"
	"github.com/hillrnate/pritunl-zero/session"
	"github.com/hillrnate/pritunl-zero/signature"
	"github.com/hillrnate/pritunl-zero/user"
	"gopkg.in/mgo.v2/bson"
	"net/http"
)

type Authorizer struct {
	typ  string
	cook *cookie.Cookie
	sess *session.Session
	sig  *signature.Signature
	srvc *service.Service
}

func (a *Authorizer) IsApi() bool {
	return a.sig != nil
}

func (a *Authorizer) IsValid() bool {
	return a.sess != nil || a.sig != nil
}

func (a *Authorizer) Clear(db *database.Database, w http.ResponseWriter,
	r *http.Request) (err error) {

	a.sess = nil
	a.sig = nil

	if a.cook != nil {
		err = a.cook.Remove(db)
		if err != nil {
			return
		}
	}

	switch a.typ {
	case Admin:
		cookie.CleanAdmin(w, r)
		break
	case Proxy:
		cookie.CleanProxy(w, r)
		break
	case User:
		cookie.CleanUser(w, r)
		break
	}

	return
}

func (a *Authorizer) Remove(db *database.Database) error {
	if a.sess == nil {
		return nil
	}

	return a.sess.Remove(db)
}

func (a *Authorizer) GetUser(db *database.Database) (
	usr *user.User, err error) {

	if a.sess != nil {
		if db != nil {
			usr, err = a.sess.GetUser(db)
			if err != nil {
				switch err.(type) {
				case *database.NotFoundError:
					usr = nil
					err = nil
					break
				default:
					return
				}
			}
		}

		if usr == nil {
			a.sess = nil
		}
	} else if a.sig != nil {
		if db != nil {
			usr, err = a.sig.GetUser(db)
			if err != nil {
				switch err.(type) {
				case *database.NotFoundError:
					usr = nil
					err = nil
					break
				default:
					return
				}
			}
		}

		if usr == nil {
			a.sig = nil
		}
	}

	return
}

func (a *Authorizer) ServiceId() bson.ObjectId {
	if a.srvc != nil {
		return a.srvc.Id
	}
	return ""
}

func (a *Authorizer) GetSession() *session.Session {
	return a.sess
}

func (a *Authorizer) SessionId() string {
	if a.sess != nil {
		return a.sess.Id
	}

	return ""
}
