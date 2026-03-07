package definition

import (
	"time"

	"github.com/jacksonzamorano/strata/component"
)

var Manifest = component.ComponentManifest{
	Name:    "jmap",
	Version: "1.0.0",
}

type AccountScope struct {
	AccountID string
}
type MailboxScope struct {
	AccountID string
	Mailbox   string
	After     time.Time
}

var AddConnection = component.Define[ConnectionParams, *Connection](Manifest, "add-connection")
var GetMailboxAfter = component.Define[MailboxScope, []Email](Manifest, "get-inbox")

type ConnectionParams struct {
	Host  string
	Token string
	Email string
}

type Connection struct {
	Host  string
	Token string
	Email string

	AccountId string
	Endpoint  string

	Mailboxes []Mailbox
}

type Email struct {
	Subject string
	From    string
	Preview string
	Body    string
	Arrived time.Time
}

type Mailbox struct {
	Id   string
	Name string
}
