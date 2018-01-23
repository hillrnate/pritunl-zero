package demo

import (
	"github.com/gin-gonic/gin"
	"github.com/pritunl/pritunl-zero/agent"
	"github.com/pritunl/pritunl-zero/audit"
	"github.com/pritunl/pritunl-zero/errortypes"
	"github.com/pritunl/pritunl-zero/log"
	"github.com/pritunl/pritunl-zero/session"
	"github.com/pritunl/pritunl-zero/settings"
	"github.com/pritunl/pritunl-zero/ssh"
	"github.com/pritunl/pritunl-zero/subscription"
	"gopkg.in/mgo.v2/bson"
	"time"
)

func IsDemo() bool {
	return true
}

func Blocked(c *gin.Context) bool {
	if !IsDemo() {
		return false
	}

	errData := &errortypes.ErrorData{
		Error:   "demo_unavailable",
		Message: "Not available in demo mode",
	}
	c.JSON(400, errData)

	return true
}

var Agent = &agent.Agent{
	OperatingSystem: agent.Linux,
	Browser:         agent.Chrome,
	Ip:              "8.8.8.8",
	Isp:             "Google",
	Continent:       "North America",
	ContinentCode:   "NA",
	Country:         "United States",
	CountryCode:     "US",
	Region:          "Washington",
	RegionCode:      "WA",
	City:            "Seattle",
	Latitude:        47.611,
	Longitude:       -122.337,
}

var Audits = []*audit.Audit{
	&audit.Audit{
		Id:        bson.ObjectIdHex("5a17f9bf051a45ffacf2b352"),
		Timestamp: time.Unix(1498018860, 0),
		Type:      "admin_login",
		Fields: audit.Fields{
			"method": "local",
		},
		Agent: Agent,
	},
}

var Sessions = []*session.Session{
	&session.Session{
		Id:         "jhgRu4n3oY0iXRYmLb77Ql5jNs2o7uWM",
		Type:       session.User,
		Timestamp:  time.Unix(1498018860, 0),
		LastActive: time.Unix(1498018860, 0),
		Removed:    false,
		Agent:      Agent,
	},
}

var Sshcerts = []*ssh.Certificate{
	&ssh.Certificate{
		Id: bson.ObjectIdHex("5a180207051a45ffacf3b846"),
		AuthorityIds: []bson.ObjectId{
			bson.ObjectIdHex("5a191ca03745632d533cf597"),
		},
		Timestamp: time.Unix(1498018860, 0),
		CertificatesInfo: []*ssh.Info{
			&ssh.Info{
				Serial:  "2207385157562819502",
				Expires: time.Unix(1498105260, 0),
				Principals: []string{
					"demo",
				},
				Extensions: []string{
					"permit-X11-forwarding",
					"permit-agent-forwarding",
					"permit-port-forwarding",
					"permit-pty",
					"permit-user-rc",
				},
			},
		},
		Agent: Agent,
	},
}

var Logs = []*log.Entry{
	&log.Entry{
		Id:        bson.ObjectIdHex("5a18e6ae051a45ffac0e5b67"),
		Level:     log.Info,
		Timestamp: time.Unix(1498018860, 0),
		Message:   "router: Starting redirect server",
		Stack:     "",
		Fields: map[string]interface{}{
			"port":       80,
			"production": true,
			"protocol":   "http",
		},
	},
	&log.Entry{
		Id:        bson.ObjectIdHex("5a190b42051a45ffac129bbc"),
		Level:     log.Info,
		Timestamp: time.Unix(1498018860, 0),
		Message:   "router: Starting web server",
		Stack:     "",
		Fields: map[string]interface{}{
			"port":       443,
			"production": true,
			"protocol":   "https",
		},
	},
}

var Subscription = &subscription.Subscription{
	Active:            true,
	Status:            "active",
	Plan:              "zero",
	Quantity:          1,
	Amount:            5000,
	PeriodEnd:         time.Unix(1893499200, 0),
	TrialEnd:          time.Time{},
	CancelAtPeriodEnd: false,
	Balance:           0,
	UrlKey:            "demo",
}
