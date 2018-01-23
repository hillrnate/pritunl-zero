package auth

import (
	"bytes"
	"encoding/json"
	"github.com/dropbox/godropbox/errors"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/errortypes"
	"github.com/hillrnate/pritunl-zero/settings"
	"github.com/hillrnate/pritunl-zero/utils"
	"net/http"
	"time"
)

const (
	Azure = "azure"
)

func AzureRequest(db *database.Database, location, query string,
	provider *settings.Provider) (redirect string, err error) {

	coll := db.Tokens()

	state, err := utils.RandStr(64)
	if err != nil {
		return
	}

	secret, err := utils.RandStr(64)
	if err != nil {
		return
	}

	data, err := json.Marshal(struct {
		License     string `json:"license"`
		Callback    string `json:"callback"`
		State       string `json:"state"`
		Secret      string `json:"secret"`
		DirectoryId string `json:"directory_id"`
		AppId       string `json:"app_id"`
		AppSecret   string `json:"app_secret"`
	}{
		License:     settings.System.License,
		Callback:    location + "/auth/callback",
		State:       state,
		Secret:      secret,
		DirectoryId: provider.Tenant,
		AppId:       provider.ClientId,
		AppSecret:   provider.ClientSecret,
	})

	req, err := http.NewRequest(
		"POST",
		settings.Auth.Server+"/v1/request/azure",
		bytes.NewBuffer(data),
	)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "auth: Auth request failed"),
		}
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "auth: Auth request failed"),
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err = &errortypes.RequestError{
			errors.Wrapf(err, "auth: Auth server error %d", resp.StatusCode),
		}
		return
	}

	authData := &authData{}
	err = json.NewDecoder(resp.Body).Decode(authData)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(
				err, "auth: Failed to parse auth response",
			),
		}
		return
	}

	tokn := &Token{
		Id:        state,
		Type:      Azure,
		Secret:    secret,
		Timestamp: time.Now(),
		Provider:  provider.Id,
		Query:     query,
	}

	err = coll.Insert(tokn)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	redirect = authData.Url

	return
}
