package containerregistry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/argoproj-labs/argocd-image-updater/pkg/webhook/types"
)

// Harbor implements webhook handler for Harbor registry
type Harbor struct {
	secret string
}

// HarborWebhookPayload represents the Harbor webhook payload structure
type HarborWebhookPayload struct {
	Type      string `json:"type"`
	OccurAt   int64  `json:"occur_at"`
	Operator  string `json:"operator"`
	EventData struct {
		Resources []struct {
			Tag    string `json:"tag"`
			Digest string `json:"digest"`
		} `json:"resources"`
		Repository struct {
			Name         string `json:"name"`
			FullName     string `json:"repo_full_name"`
			RegistryName string `json:"repo_type"`
		} `json:"repository"`
	} `json:"event_data"`
}

// NewHarbor creates a new Harbor webhook handler
func NewHarbor(secret string) *Harbor {
	return &Harbor{
		secret: secret,
	}
}

// Parse implements webhook parsing for Harbor
func (h *Harbor) Parse(r *http.Request) (*types.WebhookEvent, error) {
	var payload HarborWebhookPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to decode Harbor webhook payload: %v", err)
	}

	if len(payload.EventData.Resources) == 0 {
		return nil, fmt.Errorf("no resources in webhook payload")
	}

	return &types.WebhookEvent{
		ImageName: payload.EventData.Repository.Name,
		TagName:   payload.EventData.Resources[0].Tag,
		CreatedAt: time.Unix(payload.OccurAt, 0),
		Digest:    payload.EventData.Resources[0].Digest,
		// RegistryPrefix will be set by the webhook handler since it's extracted from the URL path
	}, nil
}
