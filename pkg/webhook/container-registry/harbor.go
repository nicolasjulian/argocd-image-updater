package containerregistry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/argoproj-labs/argocd-image-updater/pkg/webhook/types"
	"github.com/sirupsen/logrus"
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
			Digest      string `json:"digest"`
			Tag         string `json:"tag"`
			ResourceURL string `json:"resource_url"`
		} `json:"resources"`
		Repository struct {
			DateCreated int64  `json:"date_created"`
			Name        string `json:"name"`
			Namespace   string `json:"namespace"`
			FullName    string `json:"repo_full_name"`
			RepoType    string `json:"repo_type"`
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

	// Check Content-Type header
	if r.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("invalid content type: %s", r.Header.Get("Content-Type"))
	}

	// Read and decode the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}
	defer r.Body.Close()

	logrus.Debugf("Received payload: %s", string(body))

	if err := json.Unmarshal(body, &payload); err != nil {
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
