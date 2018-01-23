package authority

import (
	"crypto/rand"
	"crypto/subtle"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/hillrnate/pritunl-zero/database"
	"github.com/hillrnate/pritunl-zero/errortypes"
	"github.com/hillrnate/pritunl-zero/requires"
	"github.com/hillrnate/pritunl-zero/settings"
	"github.com/hillrnate/pritunl-zero/user"
	"github.com/hillrnate/pritunl-zero/utils"
	"golang.org/x/crypto/ssh"
	"gopkg.in/mgo.v2/bson"
	"hash/fnv"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"
)

var (
	client = &http.Client{
		Timeout: 10 * time.Second,
	}
)

type validateData struct {
	PublicKey string `bson:"public_key" json:"public_key"`
}

type Info struct {
	KeyAlg string `bson:"key_alg" json:"key_alg"`
}

type Authority struct {
	Id                 bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Name               string        `bson:"name" json:"name"`
	Type               string        `bson:"type" json:"type"`
	Info               *Info         `bson:"info" json:"info"`
	MatchRoles         bool          `bson:"match_roles" json:"match_roles"`
	Roles              []string      `bson:"roles" json:"roles"`
	Expire             int           `bson:"expire" json:"expire"`
	HostExpire         int           `bson:"host_expire" json:"host_expire"`
	PrivateKey         string        `bson:"private_key" json:"-"`
	PublicKey          string        `bson:"public_key" json:"public_key"`
	HostDomain         string        `bson:"host_domain" json:"host_domain"`
	HostProxy          string        `bson:"host_proxy" json:"host_proxy"`
	HostCertificates   bool          `bson:"host_certificates" json:"host_certificates"`
	StrictHostChecking bool          `bson:"strict_host_checking" json:"strict_host_checking"`
	HostTokens         []string      `bson:"host_tokens" json:"host_tokens"`
}

func (a *Authority) GetDomain(hostname string) string {
	return hostname + "." + a.HostDomain
}

func (a *Authority) GenerateRsaPrivateKey() (err error) {
	privKeyBytes, pubKeyBytes, err := GenerateRsaKey()
	if err != nil {
		return
	}

	a.Info = &Info{
		KeyAlg: "RSA 4096",
	}
	a.PrivateKey = strings.TrimSpace(string(privKeyBytes))
	a.PublicKey = strings.TrimSpace(string(pubKeyBytes))

	return
}

func (a *Authority) GenerateEcPrivateKey() (err error) {
	privKeyBytes, pubKeyBytes, err := GenerateEcKey()
	if err != nil {
		return
	}

	a.Info = &Info{
		KeyAlg: "EC P384",
	}
	a.PrivateKey = strings.TrimSpace(string(privKeyBytes))
	a.PublicKey = strings.TrimSpace(string(pubKeyBytes))

	return
}

func (a *Authority) GetHostDomain() string {
	if a.HostDomain == "" {
		return ""
	}

	domain := "*." + a.HostDomain

	if a.HostProxy != "" {
		hostProxy := strings.SplitN(a.HostProxy, "@", 2)
		domain += " !" + hostProxy[len(hostProxy)-1]
	}

	return domain
}

func (a *Authority) GetCertAuthority() string {
	if a.HostDomain == "" {
		return ""
	}
	return fmt.Sprintf("@cert-authority *.%s %s", a.HostDomain, a.PublicKey)
}

func (a *Authority) UserHasAccess(usr *user.User) bool {
	if !a.MatchRoles {
		return true
	}
	return usr.RolesMatch(a.Roles)
}

func (a *Authority) HostnameValidate(hostname string, port int,
	pubKey string) bool {

	domain := a.GetDomain(hostname)

	ipsNet, err := net.LookupIP(domain)
	if err != nil {
		err = &errortypes.RequestError{
			errors.Wrap(err, "authority: Failed to lookup host"),
		}

		logrus.WithFields(logrus.Fields{
			"error": err,
		}).Error("authority: Failed to lookup host")

		return false
	}

	ips := []net.IP{}
	for _, ip := range ipsNet {
		if ip.To4() != nil {
			ips = append(ips, ip)
		}
	}

	if len(ips) == 0 {
		logrus.WithFields(logrus.Fields{
			"host": domain,
		}).Error("authority: No IPv4 addresses found for host")
		return false
	}

	valid := false
	url := ""
	if port == 0 {
		port = 9748
	}

	for _, ip := range ips {
		url = fmt.Sprintf("http://%s:%d/challenge", ip, port)
		req, e := http.NewRequest(
			"GET",
			url,
			nil,
		)
		if e != nil {
			err = &errortypes.RequestError{
				errors.Wrap(e, "authority: Validation request failed"),
			}
			continue
		}

		resp, e := client.Do(req)
		if e != nil {
			err = &errortypes.RequestError{
				errors.Wrap(e, "authority: Validation request failed"),
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			err = &errortypes.RequestError{
				errors.Newf("authority: Validation request bad status %d",
					resp.StatusCode),
			}
			continue
		}

		data := &validateData{}
		e = json.NewDecoder(resp.Body).Decode(data)
		if e != nil {
			err = &errortypes.ParseError{
				errors.Wrap(e, "authority: Failed to parse response"),
			}
			break
		}

		hostPubKey := strings.TrimSpace(data.PublicKey)
		if len(hostPubKey) > settings.System.SshPubKeyLen {
			err = errortypes.ParseError{
				errors.New("authority: Public key too long"),
			}
			break
		}

		if subtle.ConstantTimeCompare([]byte(pubKey),
			[]byte(hostPubKey)) != 1 {

			err = errortypes.AuthenticationError{
				errors.New("authority: Public key does not match"),
			}
			break
		}

		valid = true
		err = nil
		break
	}

	if err != nil || !valid {
		logrus.WithFields(logrus.Fields{
			"host":  domain,
			"url":   url,
			"error": err,
		}).Error("authority: Host validation failed")
		return false
	}

	return true
}

func (a *Authority) CreateCertificate(usr *user.User, sshPubKey string) (
	cert *ssh.Certificate, certMarshaled string, err error) {

	privateKey, err := ParsePemKey(a.PrivateKey)
	if err != nil {
		return
	}

	pubKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(sshPubKey))
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to parse ssh public key"),
		}
		return
	}

	serialHash := fnv.New64a()
	serialHash.Write([]byte(bson.NewObjectId().Hex()))
	serial := serialHash.Sum64()

	expire := a.Expire
	if expire == 0 {
		expire = 600
	}
	validAfter := time.Now().Add(-5 * time.Minute).Unix()
	validBefore := time.Now().Add(
		time.Duration(expire) * time.Minute).Unix()

	if len(usr.Roles) == 0 {
		err = &errortypes.AuthenticationError{
			errors.Wrap(err, "authority: User has no roles"),
		}
		return
	}

	cert = &ssh.Certificate{
		Key:             pubKey,
		Serial:          serial,
		CertType:        ssh.UserCert,
		KeyId:           usr.Id.Hex(),
		ValidPrincipals: usr.Roles,
		ValidAfter:      uint64(validAfter),
		ValidBefore:     uint64(validBefore),
		Permissions: ssh.Permissions{
			Extensions: map[string]string{
				"permit-X11-forwarding":   "",
				"permit-agent-forwarding": "",
				"permit-port-forwarding":  "",
				"permit-pty":              "",
				"permit-user-rc":          "",
			},
		},
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return
	}

	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		return
	}

	certMarshaled = string(MarshalCertificate(cert, comment))

	return
}

func (a *Authority) CreateHostCertificate(hostname string, sshPubKey string) (
	cert *ssh.Certificate, certMarshaled string, err error) {

	privateKey, err := ParsePemKey(a.PrivateKey)
	if err != nil {
		return
	}

	pubKey, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(sshPubKey))
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to parse ssh public key"),
		}
		return
	}

	serialHash := fnv.New64a()
	serialHash.Write([]byte(bson.NewObjectId().Hex()))
	serial := serialHash.Sum64()

	expire := a.HostExpire
	if expire == 0 {
		expire = 600
	}
	validAfter := time.Now().Add(-5 * time.Minute).Unix()
	validBefore := time.Now().Add(
		time.Duration(expire) * time.Minute).Unix()

	cert = &ssh.Certificate{
		Key:             pubKey,
		Serial:          serial,
		CertType:        ssh.HostCert,
		KeyId:           hostname,
		ValidPrincipals: []string{a.GetDomain(hostname)},
		ValidAfter:      uint64(validAfter),
		ValidBefore:     uint64(validBefore),
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return
	}

	err = cert.SignCert(rand.Reader, signer)
	if err != nil {
		return
	}

	certMarshaled = string(MarshalCertificate(cert, comment))

	return
}

func (a *Authority) TokenNew() (err error) {
	if a.HostTokens == nil {
		a.HostTokens = []string{}
	}

	token, err := utils.RandStr(48)
	if err != nil {
		return
	}

	a.HostTokens = append(a.HostTokens, token)

	return
}

func (a *Authority) TokenDelete(token string) (err error) {
	if a.HostTokens == nil {
		a.HostTokens = []string{}
	}

	for i, tokn := range a.HostTokens {
		if tokn == token {
			a.HostTokens = append(
				a.HostTokens[:i], a.HostTokens[i+1:]...)
			break
		}
	}

	return
}

func (a *Authority) Export(passphrase string) (encKey string, err error) {
	block, _ := pem.Decode([]byte(a.PrivateKey))

	encBlock, err := x509.EncryptPEMBlock(
		rand.Reader,
		block.Type,
		block.Bytes,
		[]byte(passphrase),
		x509.PEMCipherAES256,
	)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "authority: Failed to encrypt private key"),
		}
		return
	}

	encodedBlock := pem.EncodeToMemory(encBlock)

	encKey = string(encodedBlock)

	return
}

func (a *Authority) Validate(db *database.Database) (
	errData *errortypes.ErrorData, err error) {

	if a.Type == "" {
		a.Type = Local
	}

	if !a.MatchRoles {
		a.Roles = []string{}
	}

	if a.PrivateKey == "" {
		err = a.GenerateRsaPrivateKey()
		if err != nil {
			return
		}
	}

	if a.Expire < 1 {
		a.Expire = 600
	} else if a.Expire > 1440 {
		a.Expire = 1440
	}

	if a.HostExpire < 1 {
		a.HostExpire = 600
	} else if a.HostExpire > 1440 {
		a.HostExpire = 1440
	} else if a.HostExpire < 15 {
		a.HostExpire = 15
	}

	if a.HostCertificates && a.HostDomain == "" {
		errData = &errortypes.ErrorData{
			Error:   "host_domain_required",
			Message: "Host domain must be set for host certificates",
		}
		return
	}

	if a.HostDomain == "" {
		a.HostCertificates = false
		a.StrictHostChecking = false
		a.HostProxy = ""
	}

	if a.HostTokens == nil || !a.HostCertificates {
		a.HostTokens = []string{}
	}

	a.Format()

	return
}

func (a *Authority) Format() {
	roles := []string{}
	rolesSet := set.NewSet()

	for _, role := range a.Roles {
		rolesSet.Add(role)
	}

	for role := range rolesSet.Iter() {
		roles = append(roles, role.(string))
	}

	sort.Strings(roles)

	a.Roles = roles

	sort.Strings(a.HostTokens)
}

func (a *Authority) Commit(db *database.Database) (err error) {
	coll := db.Authorities()

	err = coll.Commit(a.Id, a)
	if err != nil {
		return
	}

	return
}

func (a *Authority) CommitFields(db *database.Database, fields set.Set) (
	err error) {

	coll := db.Authorities()

	err = coll.CommitFields(a.Id, a, fields)
	if err != nil {
		return
	}

	return
}

func (a *Authority) Insert(db *database.Database) (err error) {
	coll := db.Authorities()

	if a.Id != "" {
		err = &errortypes.DatabaseError{
			errors.New("authority: Authority already exists"),
		}
		return
	}

	err = coll.Insert(a)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}

func init() {
	module := requires.New("authority")
	module.After("settings")

	module.Handler = func() (err error) {
		db := database.GetDatabase()
		defer db.Close()

		authrs, err := GetAll(db)
		if err != nil {
			return
		}

		for _, authr := range authrs {
			if !authr.HostCertificates && authr.HostDomain != "" &&
				authr.HostTokens != nil && len(authr.HostTokens) > 0 {

				authr.HostCertificates = true
				err = authr.CommitFields(db, set.NewSet("host_certificates"))
				if err != nil {
					return
				}
			}
		}

		return
	}
}
