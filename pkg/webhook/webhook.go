package webhook

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/argoproj-labs/argocd-image-updater/pkg/webhook/container-registry"
	"github.com/argoproj-labs/argocd-image-updater/pkg/webhook/types"
	"github.com/argoproj-labs/argocd-image-updater/registry-scanner/pkg/log"
	"github.com/argoproj-labs/argocd-image-updater/registry-scanner/pkg/registry"
	"github.com/argoproj-labs/argocd-image-updater/registry-scanner/pkg/tag"
)

var webhookEventCh = make(chan types.WebhookEvent)

// GetWebhookEventChan returns a chan for WebhookEvent
func GetWebhookEventChan() chan types.WebhookEvent {
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

	manifest, err := regClient.ManifestForTag(tagStr)
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
	var err error
	var webhookEvent *types.WebhookEvent

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid webhook path", http.StatusBadRequest)
		return
	}
	regPrefix := parts[3]
	hookSecret := getWebhookSecretByPrefix(regPrefix)

	// Support only Harbor webhook
	if r.Header.Get("X-Harbor-Event-Id") != "" {
		webhookID := r.Header.Get("X-Harbor-Event-Id")
		log.Debugf("Callback from Harbor, X-Harbor-Event-Id=%s", webhookID)
		harborHook := containerregistry.NewHarborWebhook(hookSecret)
		webhookEvent, err = harborHook.Parse(r)
		if err != nil {
			log.Errorf("Could not parse Harbor payload %v", err)
			http.Error(w, "Could not parse Harbor payload", http.StatusBadRequest)
			return
		}

		// Get tag metadata
		tagInfo, err := getTagMetadata(regPrefix, webhookEvent.ImageName, webhookEvent.TagName)
		if err != nil {
			log.Errorf("Could not get tag metadata: %v", err)
			http.Error(w, "Could not get tag metadata", http.StatusInternalServerError)
			return
		}

		// Update webhook event with metadata
		webhookEvent.CreatedAt = tagInfo.CreatedAt
		webhookEvent.Digest = tagInfo.EncodedDigest()
		webhookEvent.RegistryPrefix = regPrefix

		// Send the event to the channel
		webhookEventCh <- *webhookEvent
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

	// Return success
	w.WriteHeader(http.StatusOK)
}
