package auth

import (
	"github.com/hillrnate/pritunl-zero/database"
	"gopkg.in/mgo.v2/bson"
	"net/http"
	"time"
)

var (
	client = &http.Client{
		Timeout: 20 * time.Second,
	}
)

type authData struct {
	Url string `json:"url"`
}

type Token struct {
	Id        string        `bson:"_id"`
	Type      string        `bson:"type"`
	Secret    string        `bson:"secret"`
	Timestamp time.Time     `bson:"timestamp"`
	Provider  bson.ObjectId `bson:"provider,omitempty"`
	Query     string        `bson:"query"`
}

func (t *Token) Remove(db *database.Database) (err error) {
	coll := db.Tokens()

	err = coll.RemoveId(t.Id)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}
