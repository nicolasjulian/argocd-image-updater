package webhook

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	containerregistry "github.com/argoproj-labs/argocd-image-updater/pkg/webhook/container-registry"
	"github.com/argoproj-labs/argocd-image-updater/pkg/webhook/types"
	"github.com/argoproj-labs/argocd-image-updater/registry-scanner/pkg/env"
	"github.com/argoproj-labs/argocd-image-updater/registry-scanner/pkg/log"
)

var webhookEventCh = make(chan types.WebhookEvent)
var ErrInvalidSecret = errors.New("invalid secret")

func GetWebhookEventChan() chan types.WebhookEvent {
	return webhookEventCh
}

func StartRegistryHookServer(port int) chan error {
	errCh := make(chan error)
	go func() {
		sm := http.NewServeMux()
		sm.HandleFunc("/api/webhook/harbor", handleWebhook)
		errCh <- http.ListenAndServe(fmt.Sprintf(":%d", port), sm)
	}()
	return errCh
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Check for Harbor webhook
	if r.Header.Get("authorization") != "" {
		harborSecret := env.GetStringVal("WEBHOOK_HARBOR_SECRET", "")
		harborHook := containerregistry.NewHarbor(harborSecret)
		event, err := harborHook.Parse(r)
		if err != nil {
			if err == ErrInvalidSecret {
				log.Errorf("Invalid Harbor webhook secret: %v", err)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Invalid credentials"))
			} else {
				log.Errorf("Error parsing Harbor webhook: %v", err)
				w.WriteHeader(http.StatusBadRequest)
			}
			return
		}

		 // Extract registry prefix from the URL path
        parts := strings.Split(r.URL.Path, "/")
        if len(parts) < 4 {
            log.Errorf("Invalid URL path: %s", r.URL.Path)
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        regPrefix := parts[3]
        event.RegistryPrefix = regPrefix

        log.Infof("Received webhook event: registry=%s, image=%s, tag=%s", regPrefix, event.ImageName, event.TagName)

		// Send the event to the channel for processing
		webhookEventCh <- *event
		w.WriteHeader(http.StatusOK)
		return
	} else {
		log.Errorf("Authorization header not found")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Authentication required"))
		return
	}
}
