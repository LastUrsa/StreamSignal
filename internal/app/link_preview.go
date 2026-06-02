package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"StreamSignal/internal/domain"
	"StreamSignal/internal/platforms/netguard"
	"golang.org/x/net/html"
)

const maxLinkPreviewBytes = 1_000_000

type linkPreviewFetcher func(context.Context, string) (*domain.LinkPreview, error)

func fetchLinkPreview(ctx context.Context, rawURL string) (*domain.LinkPreview, error) {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return nil, nil
	}
	if err := netguard.ValidatePublicHTTPURL(ctx, trimmedURL, "Link preview"); err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 4 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	currentURL := trimmedURL
	for redirects := 0; redirects < 3; redirects++ {
		preview, redirectURL, err := fetchLinkPreviewOnce(ctx, client, currentURL)
		if err != nil {
			return nil, err
		}
		if redirectURL == "" {
			return preview, nil
		}
		if err := netguard.ValidatePublicHTTPURL(ctx, redirectURL, "Link preview redirect"); err != nil {
			return nil, err
		}
		currentURL = redirectURL
	}

	return nil, errors.New("link preview followed too many redirects")
}

func fetchLinkPreviewOnce(ctx context.Context, client *http.Client, rawURL string) (*domain.LinkPreview, string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build link preview request: %w", err)
	}
	request.Header.Set("User-Agent", "StreamSignal/1.0 LinkPreview")
	request.Header.Set("Accept", "text/html,application/xhtml+xml")

	response, err := client.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("fetch link preview: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 && response.StatusCode < 400 {
		location := strings.TrimSpace(response.Header.Get("Location"))
		if location == "" {
			return nil, "", fmt.Errorf("link preview redirect missing location: %s", response.Status)
		}
		redirectURL, err := response.Request.URL.Parse(location)
		if err != nil {
			return nil, "", fmt.Errorf("parse link preview redirect: %w", err)
		}
		return nil, redirectURL.String(), nil
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", fmt.Errorf("link preview fetch failed: %s", response.Status)
	}

	mediaType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if mediaType != "" && !strings.EqualFold(mediaType, "text/html") && !strings.EqualFold(mediaType, "application/xhtml+xml") {
		return nil, "", fmt.Errorf("link preview content must be HTML")
	}

	limitedBody := io.LimitReader(response.Body, maxLinkPreviewBytes+1)
	data, err := io.ReadAll(limitedBody)
	if err != nil {
		return nil, "", fmt.Errorf("read link preview: %w", err)
	}
	if len(data) > maxLinkPreviewBytes {
		return nil, "", fmt.Errorf("link preview page is too large")
	}

	preview := parseLinkPreview(response.Request.URL, strings.NewReader(string(data)))
	if preview.Title == "" && preview.Description == "" && preview.ImageURL == "" {
		return nil, "", nil
	}
	if preview.ImageURL != "" {
		if err := netguard.ValidatePublicHTTPURL(ctx, preview.ImageURL, "Link preview image"); err != nil {
			preview.ImageURL = ""
		}
	}
	return preview, "", nil
}

func parseLinkPreview(pageURL *url.URL, reader io.Reader) *domain.LinkPreview {
	preview := &domain.LinkPreview{URL: pageURL.String()}
	tokenizer := html.NewTokenizer(reader)

	for {
		tokenType := tokenizer.Next()
		switch tokenType {
		case html.ErrorToken:
			return normalizeLinkPreview(preview)
		case html.StartTagToken, html.SelfClosingTagToken:
			token := tokenizer.Token()
			switch strings.ToLower(token.Data) {
			case "title":
				if preview.Title == "" {
					preview.Title = readTitleText(tokenizer)
				}
			case "meta":
				applyMetaTag(pageURL, preview, token.Attr)
			}
		}
	}
}

func applyMetaTag(pageURL *url.URL, preview *domain.LinkPreview, attrs []html.Attribute) {
	var key, content string
	for _, attr := range attrs {
		switch strings.ToLower(attr.Key) {
		case "property", "name":
			key = strings.ToLower(strings.TrimSpace(attr.Val))
		case "content":
			content = strings.TrimSpace(attr.Val)
		}
	}
	if key == "" || content == "" {
		return
	}

	switch key {
	case "og:title", "twitter:title":
		if preview.Title == "" {
			preview.Title = content
		}
	case "og:description", "twitter:description", "description":
		if preview.Description == "" {
			preview.Description = content
		}
	case "og:image", "twitter:image", "twitter:image:src":
		if preview.ImageURL == "" {
			preview.ImageURL = resolvePreviewURL(pageURL, content)
		}
	case "og:site_name":
		if preview.SiteName == "" {
			preview.SiteName = content
		}
	case "og:url":
		if resolved := resolvePreviewURL(pageURL, content); resolved != "" {
			preview.URL = resolved
		}
	}
}

func readTitleText(tokenizer *html.Tokenizer) string {
	if tokenizer.Next() != html.TextToken {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(string(tokenizer.Text())))
}

func resolvePreviewURL(base *url.URL, rawValue string) string {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	return base.ResolveReference(parsed).String()
}

func normalizeLinkPreview(preview *domain.LinkPreview) *domain.LinkPreview {
	preview.Title = compactPreviewText(preview.Title, 160)
	preview.Description = compactPreviewText(preview.Description, 280)
	preview.SiteName = compactPreviewText(preview.SiteName, 80)
	return preview
}

func compactPreviewText(value string, maxLength int) string {
	compact := strings.Join(strings.Fields(value), " ")
	if len(compact) <= maxLength {
		return compact
	}
	return compact[:maxLength-3] + "..."
}
