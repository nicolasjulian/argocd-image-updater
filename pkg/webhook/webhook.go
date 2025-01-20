package webhook

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/argoproj-labs/argocd-image-updater/registry-scanner/pkg/log"
	"github.com/argoproj-labs/argocd-image-updater/registry-scanner/pkg/registry"
	"github.com/argoproj-labs/argocd-image-updater/registry-scanner/pkg/tag"
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
	New(secret string) (RegistryWebhook, error)
	Parse(r *http.Request, events ...Event) (*WebhookEvent, error)
}

var webhookEventCh (chan WebhookEvent) = make(chan WebhookEvent)

// GetWebhookEventChan return a chan for WebhookEvent
func GetWebhookEventChan() chan WebhookEvent {
	return webhookEventCh
}

// StartRegistryHookServer starts a new HTTP server for registry hook on given port
func StartRegistryHookServer(port int) chan error {
	errCh := make(chan error)
	go func() {
		sm := http.NewServeMux()

		regPrefixes := registry.ConfiguredEndpoints()
		for _, prefix := range regPrefixes {
			var regPrefix string = prefix
			if regPrefix == "" {
				regPrefix = "docker.io"
			}
			var path string = fmt.Sprintf("/api/webhook/%s", regPrefix)
			sm.HandleFunc(path, webhookHandler)
		}
		errCh <- http.ListenAndServe(fmt.Sprintf(":%d", port), sm)
	}()
	return errCh
}

func getTagMetadata(regPrefix string, imageName string, tagStr string) (*tag.TagInfo, error) {
	rep, err := registry.GetRegistryEndpoint(regPrefix)
	if err != nil {
		log.Errorf("Could not get registry endpoint for %s", regPrefix)
		return nil, err
	}

	regClient, err := registry.NewClient(rep, rep.Username, rep.Password)
	if err != nil {
		log.Errorf("Could not creating new registry client for %s", regPrefix)
		return nil, err
	}

	var nameInRegistry string
	if len := len(strings.Split(imageName, "/")); len == 1 && rep.DefaultNS != "" {
		nameInRegistry = rep.DefaultNS + "/" + imageName
		log.Debugf("Using canonical image name '%s' for image '%s'", nameInRegistry, imageName)
	} else {
		nameInRegistry = imageName
	}

	err = regClient.NewRepository(nameInRegistry)
	if err != nil {
		log.Errorf("Could not create new repository for %s", nameInRegistry)
		return nil, err
	}

	manifest, err := regClient.GetManifest(nameInRegistry, tagStr)
	if err != nil {
		log.Errorf("Could not get manifest for %s:%s - %v", nameInRegistry, tagStr, err)
		return nil, err
	}
	tagInfo, err := regClient.TagMetadata(manifest, nil)

	if err != nil {
		log.Errorf("Could not fetch metadata for %s:%s - no metadata returned by registry: %v", regPrefix, tagStr, err)
		return nil, err
	}

	return tagInfo, nil
}

func getWebhookSecretByPrefix(regPrefix string) string {
	rep, err := registry.GetRegistryEndpoint(regPrefix)
	if err != nil {
		log.Errorf("Could not get registry endpoint %s", regPrefix)
		return ""
	}
	return rep.HookSecret
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	var webhookEv *WebhookEvent
	var err error

	parts := strings.Split(r.URL.Path, "/")
	regPrefix := parts[3]
	hookSecret := getWebhookSecretByPrefix(regPrefix)

	// Support only Harbor webhook
	if r.Header.Get("X-Harbor-Event-Id") != "" {
		webhookID := r.Header.Get("X-Harbor-Event-Id")
		log.Debugf("Callback from Harbor, X-Harbor-Event-Id=%s", webhookID)
		harborHook := NewHarborWebhook(hookSecret)
		webhookEv, err = harborHook.Parse(r)
		if err != nil {
			log.Errorf("Could not parse Harbor payload %v", err)
			http.Error(w, "Could not parse Harbor payload", http.StatusBadRequest)
			return
		}
	} else {
		log.Debugf("Ignoring unknown webhook event")
		http.Error(w, "Unknown webhook event", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Infof("Webhook processing failed: %s", err)
		status := http.StatusBadRequest
		if r.Method != "POST" {
			status = http.StatusMethodNotAllowed
		}
		http.Error(w, "Webhook processing failed", status)
		return
	}

	// Process the webhook event
	// ...
}
