# Strata JMAP Component

`strata-jmap` is a reusable JMAP mail component for [Strata](https://github.com/jacksonzamorano/strata).

It provides a typed component contract plus a component binary that a Strata app can launch out of process. The current implementation authenticates with a bearer token, discovers the JMAP session from `/.well-known/jmap`, lists available mailboxes, and fetches recent email messages from a mailbox after a given timestamp.

## What It Exposes

This component currently exports two typed actions from the `definition` package:

- `add-connection`: Connect to a JMAP server and return the discovered account metadata plus available mailboxes.
- `get-inbox`: Fetch up to 20 emails from a mailbox received after a supplied `time.Time`.

Your Strata app should import `github.com/jacksonzamorano/strata-jmap/definition` so component calls stay typed.

## Project Layout

- `main.go`: component entrypoint and mounted handlers
- `api.go`: JMAP session, mailbox, and email API calls
- `definition/definition.go`: manifest, request types, response types, and exported component definitions

## Using It In A Strata App

Import this project into your Strata runtime, then call it through the shared definitions package.

Example runtime setup:

```go
package main

import "github.com/jacksonzamorano/strata"

func main() {
	rt := strata.NewRuntime(
		[]strata.Task{
			// your tasks here
		},
		strata.Import(
			strata.ImportLocal("/path/to/strata-jmap"),
		),
	)

	panic(rt.Start())
}
```

Example task usage:

```go
package main

import (
	"time"

	jmap "github.com/jacksonzamorano/strata-jmap/definition"
	"github.com/jacksonzamorano/strata"
)

func syncInbox(input strata.RouteTaskNoInput, ctx *strata.TaskContext) *strata.RouteResult {
	conn, ok := jmap.AddConnection.Execute(ctx.Container, jmap.ConnectionParams{
		Host:  "https://mail.example.com",
		Token: "your-jmap-bearer-token",
		Email: "you@example.com",
	})
	if !ok || conn == nil || len(conn.Mailboxes) == 0 {
		return strata.RouteResultSuccess("could not connect")
	}

	emails, ok := jmap.GetMailboxAfter.Execute(ctx.Container, jmap.MailboxScope{
		AccountID: conn.AccountId,
		Mailbox:   conn.Mailboxes[0].Id,
		After:     time.Now().Add(-24 * time.Hour),
	})
	if !ok {
		return strata.RouteResultSuccess("could not fetch mailbox")
	}

	return strata.RouteResultSuccess(emails)
}
```

For the Strata runtime model and component import workflow, see `jacksonzamorano/strata` on GitHub.

## Component Contract

`ConnectionParams`:

- `Host`: Base server URL. The component requests `HOST/.well-known/jmap` to discover the API endpoint.
- `Token`: Bearer token used for JMAP requests.
- `Email`: Stored on the returned connection object for caller convenience.

`Connection` includes:

- `AccountId`: The first discovered JMAP account ID.
- `Endpoint`: The resolved JMAP API URL from the session document.
- `Mailboxes`: A simplified mailbox list with `Id` and `Name`.

`MailboxScope` includes:

- `AccountID`: Account to query.
- `Mailbox`: Mailbox ID to fetch from.
- `After`: Only return messages after this timestamp.

`Email` includes:

- `Subject`
- `From`
- `Preview`
- `Body`
- `Arrived`

## Running And Developing

This repository is a normal Go module:

```bash
go build ./...
go test ./...
```

When imported with `strata.ImportLocal(...)`, Strata builds the component project and launches the resulting binary for you.

## Current Notes

- The component manifest name is `jmap` and the current version is `1.0.0`.
- Connections are kept in an in-memory slice in `main.go`, so they are not persisted across component restarts.
- `get-inbox` currently reads the plain text body when present and returns at most 20 messages.
- The current auth flow uses bearer tokens rather than basic auth.
