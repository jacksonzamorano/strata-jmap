package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	d "github.com/jacksonzamorano/strata-jmap/definition"
	"github.com/jacksonzamorano/strata/component"
)

var client http.Client

type Request struct {
	Using       []string `json:"using"`
	MethodCalls [][]any  `json:"methodCalls"`
}

type Session struct {
	APIURL    string             `json:"apiUrl"`
	AccountID string             // populated after parsing
	Accounts  map[string]Account `json:"accounts"`
}
type Account struct {
	Name string `json:"name"`
}

func GetSession(cn *d.Connection) error {
	req, _ := http.NewRequest("GET", cn.Host+"/.well-known/jmap", nil)
	// req.SetBasicAuth(cn.Email, cn.Token)
	req.Header.Add("Authorization", "Bearer "+cn.Token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		e, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(e))
	}

	var s Session
	json.NewDecoder(resp.Body).Decode(&s)

	for id := range s.Accounts {
		cn.AccountId = id
		break
	}
	cn.Endpoint = s.APIURL
	return nil
}

type QueryResult struct {
	IDs []string `json:"ids"`
}

type EmailResult struct {
	List []Email `json:"list"`
}

type Email struct {
	ID         string                    `json:"id"`
	Subject    string                    `json:"subject"`
	From       []Address                 `json:"from"`
	ReceivedAt string                    `json:"receivedAt"`
	Preview    string                    `json:"preview"`
	TextBody   []EmailBodyPart           `json:"textBody"`
	HTMLBody   []EmailBodyPart           `json:"htmlBody,omitempty"`
	BodyValues map[string]EmailBodyValue `json:"bodyValues"`
}
type EmailBodyPart struct {
	PartID string `json:"partId"`
	Type   string `json:"type"`
}

type EmailBodyValue struct {
	Value             string `json:"value"`
	IsEncodingProblem bool   `json:"isEncodingProblem"`
	IsTruncated       bool   `json:"isTruncated"`
}
type Address struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Response struct {
	MethodResponses [][]json.RawMessage `json:"methodResponses"`
}

type Mailbox struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Role         string `json:"role"`
	TotalEmails  int    `json:"totalEmails"`
	UnreadEmails int    `json:"unreadEmails"`
}

type MailboxResult struct {
	List []Mailbox `json:"list"`
}

func fetchMailboxes(session *d.Connection, c *component.ComponentContainer) ([]Mailbox, error) {
	resp, err := call(session, c, Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: [][]any{
			{"Mailbox/get", map[string]any{
				"accountId": session.AccountId,
				"ids":       nil, // nil = fetch all
			}, "m0"},
		},
	})
	if err != nil {
		return nil, err
	}

	var result MailboxResult
	json.Unmarshal(resp.MethodResponses[0][1], &result)
	return result.List, nil
}

func call(cn *d.Connection, c *component.ComponentContainer, req Request) (*Response, error) {
	body, _ := json.Marshal(req)
	// c.Logger.Log("Sending '%s'", string(body))
	httpReq, _ := http.NewRequest("POST", cn.Endpoint, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Add("Authorization", "Bearer "+cn.Token)

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(raw))
	}

	var out Response
	json.Unmarshal(raw, &out)
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(out.MethodResponses) == 0 {
		return nil, fmt.Errorf("empty methodResponses: %s", string(raw))
	}

	return &out, nil
}

func fetchMailbox(session *d.Connection, c *component.ComponentContainer, id string, after time.Time) ([]Email, error) {
	resp, err := call(session, c, Request{
		Using: []string{"urn:ietf:params:jmap:core", "urn:ietf:params:jmap:mail"},
		MethodCalls: [][]any{
			// 1. Query inbox
			{"Email/query", map[string]any{
				"accountId": session.AccountId,
				"filter": map[string]any{
					"operator": "AND",
					"conditions": []any{
						map[string]any{"inMailbox": id},
						map[string]any{"after": after.Format(time.RFC3339)},
					},
				}, // adjust for your server
				"sort":  []map[string]any{{"property": "receivedAt", "isAscending": false}},
				"limit": 20,
			}, "q0"},
			// 2. Fetch details using IDs from step 1
			{"Email/get", map[string]any{
				"accountId": session.AccountId,
				"#ids": map[string]any{
					"resultOf": "q0",
					"name":     "Email/query",
					"path":     "/ids",
				},
				"properties":          []string{"id", "subject", "from", "preview", "receivedAt", "textBody", "htmlBody", "bodyValues"},
				"bodyProperties":      []string{"partId", "type"},
				"fetchTextBodyValues": true,
				"fetchHTMLBodyValues": true,
			}, "g0"},
		},
	})
	if err != nil {
		return nil, err
	}

	// Parse Email/get response (second method response)
	var emailResult EmailResult
	json.Unmarshal(resp.MethodResponses[1][1], &emailResult)
	return emailResult.List, nil
}
