package user

import (
	"github.com/dropbox/godropbox/errors"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/errortypes"
	"github.com/hillrnate/pritunl-zero/utils"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
)

func Get(db *database.Database, userId bson.ObjectId) (
	usr *User, err error) {

	coll := db.Users()
	usr = &User{}

	err = coll.FindOneId(userId, usr)
	if err != nil {
		return
	}

	return
}

func GetUpdate(db *database.Database, userId bson.ObjectId) (
	usr *User, err error) {

	coll := db.Users()
	usr = &User{}
	timestamp := time.Now()

	change := mgo.Change{
		Update: &bson.M{
			"$set": &bson.M{
				"last_active": timestamp,
			},
		},
	}

	_, err = coll.FindId(userId).Apply(change, usr)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	usr.LastActive = timestamp

	return
}

func GetTokenUpdate(db *database.Database, token string) (
	usr *User, err error) {

	coll := db.Users()
	usr = &User{}
	timestamp := time.Now()

	change := mgo.Change{
		Update: &bson.M{
			"$set": &bson.M{
				"last_active": timestamp,
			},
		},
	}

	_, err = coll.Find(&bson.M{
		"token": token,
	}).Apply(change, usr)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	usr.LastActive = timestamp

	return
}

func GetUsername(db *database.Database, typ, username string) (
	usr *User, err error) {

	coll := db.Users()
	usr = &User{}

	if username == "" {
		err = &errortypes.NotFoundError{
			errors.New("user: Username empty"),
		}
		return
	}

	err = coll.FindOne(&bson.M{
		"type":     typ,
		"username": username,
	}, usr)
	if err != nil {
		return
	}

	return
}

func GetKeybase(db *database.Database, keybase string) (
	usr *User, err error) {

	coll := db.Users()
	usr = &User{}

	if keybase == "" {
		err = &errortypes.NotFoundError{
			errors.New("user: Keybase empty"),
		}
		return
	}

	err = coll.FindOne(&bson.M{
		"keybase": keybase,
	}, usr)
	if err != nil {
		return
	}

	return
}

func GetAll(db *database.Database, query *bson.M, page, pageCount int) (
	users []*User, count int, err error) {

	coll := db.Users()
	users = []*User{}

	qury := coll.Find(query)

	count, err = qury.Count()
	if err != nil {
		err = database.ParseError(err)
		return
	}

	skip := utils.Min(page*pageCount, utils.Max(0, count-pageCount))

	cursor := qury.Sort("username").Skip(skip).Limit(pageCount).Iter()

	usr := &User{}
	for cursor.Next(usr) {
		users = append(users, usr)
		usr = &User{}
	}

	err = cursor.Close()
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}

func Remove(db *database.Database, userIds []bson.ObjectId) (
	errData *errortypes.ErrorData, err error) {

	coll := db.Users()

	count, err := coll.Find(bson.M{
		"_id": &bson.M{
			"$nin": userIds,
		},
		"administrator": "super",
	}).Count()
	if err != nil {
		err = database.ParseError(err)
		return
	}

	if count == 0 {
		errData = &errortypes.ErrorData{
			Error:   "user_remove_super",
			Message: "Cannot remove all super administrators",
		}
		return
	}

	coll = db.Sessions()

	_, err = coll.RemoveAll(&bson.M{
		"user": &bson.M{
			"$in": userIds,
		},
	})
	if err != nil {
		err = database.ParseError(err)
		return
	}

	coll = db.Users()

	_, err = coll.RemoveAll(&bson.M{
		"_id": &bson.M{
			"$in": userIds,
		},
	})
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}

func Count(db *database.Database) (count int, err error) {
	coll := db.Users()

	count, err = coll.Count()
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}

func hasSuperSkip(db *database.Database, skipId bson.ObjectId) (
	exists bool, err error) {

	coll := db.Users()

	count, err := coll.Find(&bson.M{
		"_id": &bson.M{
			"$ne": skipId,
		},
		"administrator": "super",
	}).Count()
	if err != nil {
		err = database.ParseError(err)
		return
	}

	if count > 0 {
		exists = true
	}

	return
}
