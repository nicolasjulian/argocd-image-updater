package containerregistry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/argoproj-labs/argocd-image-updater/pkg/webhook/types"
)

type HarborWebhook struct {
	secret string
}

type HarborWebhookEvent struct {
	Type      string `json:"type"`
	OccurAt   int64  `json:"occur_at"`
	Operator  string `json:"operator"`
	EventData struct {
		Repository struct {
			Name         string `json:"name"`
			Namespace    string `json:"namespace"`
			RepoFullName string `json:"repo_full_name"`
			RepoType     string `json:"repo_type"`
		} `json:"repository"`
		Resources []struct {
			Digest      string `json:"digest"`
			Tag         string `json:"tag"`
			ResourceURL string `json:"resource_url"`
		} `json:"resources"`
	} `json:"event_data"`
}

func NewHarborWebhook(secret string) types.RegistryWebhook {
	return &HarborWebhook{
		secret: secret,
	}
}

func (h *HarborWebhook) Parse(r *http.Request, events ...types.Event) (*types.WebhookEvent, error) {
	if r.Method != "POST" {
		return nil, fmt.Errorf("invalid HTTP method: %s", r.Method)
	}

	// Validate webhook secret if provided
	if h.secret != "" {
		// Add Harbor-specific secret validation here if needed
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	defer r.Body.Close()

	var harborEvent HarborWebhookEvent
	if err := json.Unmarshal(body, &harborEvent); err != nil {
		return nil, fmt.Errorf("failed to parse Harbor webhook payload: %v", err)
	}

	// Only process if we have resources
	if len(harborEvent.EventData.Resources) == 0 {
		return nil, fmt.Errorf("no resources in Harbor webhook event")
	}

	// Create WebhookEvent from Harbor event
	webhookEvent := &types.WebhookEvent{
		RepoName:  harborEvent.EventData.Repository.RepoFullName,
		ImageName: harborEvent.EventData.Repository.Name,
		TagName:   harborEvent.EventData.Resources[0].Tag,
		CreatedAt: time.Unix(harborEvent.OccurAt, 0),
		Digest:    harborEvent.EventData.Resources[0].Digest,
	}

	return webhookEvent, nil
}
