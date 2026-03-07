package main

import (
	"sync"
	"time"

	d "github.com/jacksonzamorano/strata-jmap/definition"
	"github.com/jacksonzamorano/strata/component"
)

var connectionsMutex sync.RWMutex
var connections []d.Connection

func connection(acc string) *d.Connection {
	for r := range connections {
		if connections[r].AccountId == acc {
			return &connections[r]
		}
	}
	return nil
}

func addConnection(in *component.ComponentInput[d.ConnectionParams, *d.Connection], container *component.ComponentContainer) *component.ComponentReturn[*d.Connection] {
	container.Logger.Log("Connecting...")
	cn := d.Connection{
		Host:  in.Body.Host,
		Token: in.Body.Token,
		Email: in.Body.Email,
	}
	err := GetSession(&cn)
	if err != nil {
		container.Logger.Log("Could not log in: %s", err.Error())
		return in.Return(nil)
	}
	container.Logger.Log("Connected with account ID %s, fetching mailboxes", cn.AccountId)

	mboxes, err := fetchMailboxes(&cn, container)
	if err != nil {
		container.Logger.Log("Could not fetch mailboxes: %s", err.Error())
		return in.Return(nil)
	}
	for mbi := range mboxes {
		cn.Mailboxes = append(cn.Mailboxes, d.Mailbox{
			Id:   mboxes[mbi].ID,
			Name: mboxes[mbi].Name,
		})
	}

	connectionsMutex.Lock()
	defer connectionsMutex.Unlock()
	connections = append(connections, cn)
	return in.Return(&cn)
}

func getMailbox(in *component.ComponentInput[d.MailboxScope, []d.Email], container *component.ComponentContainer) *component.ComponentReturn[[]d.Email] {
	emails := []d.Email{}

	cn := connection(in.Body.AccountID)
	if cn == nil {
		return in.Return(emails)
	}

	fetchedEmails, err := fetchMailbox(cn, container, in.Body.Mailbox, in.Body.After)
	if err != nil {
		container.Logger.Log("Error when fetching inbox: %s", err.Error())
		return in.Return(emails)
	}

	for fe := range fetchedEmails {
		e := fetchedEmails[fe]
		arr, _ := time.Parse(time.RFC3339, e.ReceivedAt)

		var textBody string
		for _, part := range e.TextBody {
			if v, ok := e.BodyValues[part.PartID]; ok && v.Value != "" {
				textBody = v.Value
			}
		}

		emails = append(emails, d.Email{
			Subject: e.Subject,
			Arrived: arr,
			From:    e.From[0].Email,
			Preview: e.Preview,
			Body:    textBody,
		})
	}

	return in.Return(emails)
}

func main() {
	component.CreateComponent(d.Manifest,
		component.Mount(d.AddConnection, addConnection),
		component.Mount(d.GetMailboxAfter, getMailbox),
	).Start()
}
