package app

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"StreamSignal/internal/domain"
)

const storedSecretRefPrefix = "secret://"

func validateDestinationConfig(destination domain.Destination) error {
	switch destination.Platform {
	case domain.PlatformDiscord:
		config, err := parseDiscordDestinationConfig(destination.ConfigJSON)
		if err != nil {
			return err
		}
		if strings.TrimSpace(config.WebhookKey) == "" {
			return fmt.Errorf("Discord webhook key is required")
		}
		if isStoredSecretRef(config.WebhookKey) {
			return fmt.Errorf("Discord stored webhook could not be loaded; re-enter and save the webhook")
		}
		parsedWebhook, err := url.ParseRequestURI(strings.TrimSpace(config.WebhookKey))
		if err != nil {
			return fmt.Errorf("Discord webhook key must be a valid absolute URL")
		}
		if parsedWebhook.Scheme != "http" && parsedWebhook.Scheme != "https" {
			return fmt.Errorf("Discord webhook key must use http or https")
		}
		return nil
	case domain.PlatformBluesky:
		config, err := parseBlueskyDestinationConfig(destination.ConfigJSON)
		if err != nil {
			return err
		}
		if strings.TrimSpace(config.AccountIdentifier) == "" {
			return fmt.Errorf("Bluesky account identifier is required")
		}
		if strings.TrimSpace(config.CredentialKey) == "" {
			return fmt.Errorf("Bluesky credential key is required")
		}
		if isStoredSecretRef(config.CredentialKey) {
			return fmt.Errorf("Bluesky stored credential could not be loaded; re-enter and save the app password")
		}
		return nil
	case domain.PlatformMastodon:
		config, err := parseMastodonDestinationConfig(destination.ConfigJSON)
		if err != nil {
			return err
		}
		if strings.TrimSpace(config.CredentialKey) == "" {
			return fmt.Errorf("Mastodon credential key is required")
		}
		if isStoredSecretRef(config.CredentialKey) {
			return fmt.Errorf("Mastodon stored credential could not be loaded; re-enter and save the access token")
		}
		if strings.TrimSpace(config.InstanceURL) == "" {
			return fmt.Errorf("Mastodon instance URL is required")
		}
		if _, err := url.ParseRequestURI(strings.TrimSpace(config.InstanceURL)); err != nil {
			return fmt.Errorf("Mastodon instance URL must be a valid absolute URL")
		}
		return nil
	default:
		return fmt.Errorf("unsupported destination platform")
	}
}

func isStoredSecretRef(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), storedSecretRefPrefix)
}

func validateDestinationCredentialConfig(destination domain.Destination) error {
	switch destination.Platform {
	case domain.PlatformDiscord, domain.PlatformBluesky, domain.PlatformMastodon:
		return validateDestinationConfig(destination)
	default:
		return fmt.Errorf("unsupported destination platform")
	}
}

func parseDiscordDestinationConfig(configJSON string) (discordDestinationConfig, error) {
	var config discordDestinationConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return discordDestinationConfig{}, fmt.Errorf("invalid Discord config")
	}
	return config, nil
}

func parseBlueskyDestinationConfig(configJSON string) (blueskyDestinationConfig, error) {
	var config blueskyDestinationConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return blueskyDestinationConfig{}, fmt.Errorf("invalid Bluesky config")
	}
	return config, nil
}

func parseMastodonDestinationConfig(configJSON string) (mastodonDestinationConfig, error) {
	var config mastodonDestinationConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return mastodonDestinationConfig{}, fmt.Errorf("invalid Mastodon config")
	}
	return config, nil
}
