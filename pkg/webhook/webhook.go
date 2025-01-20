package webhook

import (
	"fmt"
	"net/http"

	containerregistry "github.com/argoproj-labs/argocd-image-updater/pkg/webhook/container-registry"
	"github.com/argoproj-labs/argocd-image-updater/pkg/webhook/types"
	"github.com/argoproj-labs/argocd-image-updater/registry-scanner/pkg/log"
)

var webhookEventCh = make(chan types.WebhookEvent)

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
	if r.Header.Get("X-Harbor-Event-Id") != "" {
		harborHook := containerregistry.NewHarbor("")
		event, err := harborHook.Parse(r)
		if err != nil {
			log.Errorf("Error parsing Harbor webhook: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Send the event to the channel for processing
		webhookEventCh <- *event
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusBadRequest)
}
