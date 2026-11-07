package sip

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// This checks local configuration only; it never validates by posting or returns
// account identifiers, tokens, webhook URLs, credential keys, or profile content.
func (s *Service) diagnosticDependencies(ctx context.Context) any {
	if s.destinations == nil {
		return map[string]any{"destinations": []any{}, "state": "unavailable"}
	}
	destinations, err := s.destinations.List(ctx)
	if err != nil {
		return map[string]any{"destinations": []any{}, "state": "unavailable", "errorCode": "UNAVAILABLE"}
	}
	results := []map[string]any{}
	for _, destination := range destinations {
		if len(results) >= 128 {
			break
		}
		config := map[string]any{}
		err := json.Unmarshal([]byte(destination.ConfigJSON), &config)
		state := "unknown"
		present := func(key string) bool { value, _ := config[key].(string); return strings.TrimSpace(value) != "" }
		if err == nil {
			configured := false
			switch string(destination.Platform) {
			case "discord":
				configured = present("webhookKey")
			case "bluesky":
				configured = present("accountIdentifier") && present("credentialKey")
			case "mastodon":
				configured = present("instanceURL") && present("credentialKey")
			}
			if configured {
				state = "configured"
			} else {
				state = "missing"
			}
		}
		lastCode, lastObserved := "", ""
		s.observer.mu.Lock()
		for _, op := range s.observer.operations {
			for _, r := range op.Results {
				if r.DestinationID == destination.ID && op.CompletedAt > lastObserved {
					lastCode, lastObserved = r.ErrorCode, op.CompletedAt
				}
			}
		}
		s.observer.mu.Unlock()
		lastAttempt, _ := time.Parse(time.RFC3339Nano, lastObserved)
		if state == "configured" && lastCode == "AUTH_REJECTED" && (destination.UpdatedAt.IsZero() || !destination.UpdatedAt.After(lastAttempt)) {
			state = "rejected"
		}
		results = append(results, map[string]any{"destinationId": destination.ID, "platform": destination.Platform, "enabled": destination.Enabled, "credentialState": state, "lastAttemptErrorCode": lastCode, "lastAttemptAt": lastObserved})
	}
	return map[string]any{"destinations": results, "state": "ready"}
}
