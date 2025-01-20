package types

import (
	"net/http"
	"time"
)

type WebhookEvent struct {
	RegistryPrefix string
	RepoName       string
	ImageName      string
	TagName        string
	CreatedAt      time.Time
	Digest         string
}

type Event string

type RegistryWebhook interface {
	Parse(r *http.Request, events ...Event) (*WebhookEvent, error)
}
