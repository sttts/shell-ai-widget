package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// duckDuckGoResponse represents the response from DuckDuckGo Instant Answer API
type duckDuckGoResponse struct {
	Abstract       string `json:"Abstract"`
	AbstractText   string `json:"AbstractText"`
	AbstractSource string `json:"AbstractSource"`
	AbstractURL    string `json:"AbstractURL"`
	Answer         string `json:"Answer"`
	AnswerType     string `json:"AnswerType"`
	Definition     string `json:"Definition"`
	DefinitionURL  string `json:"DefinitionURL"`
	Heading        string `json:"Heading"`
	RelatedTopics  []struct {
		Text     string `json:"Text"`
		FirstURL string `json:"FirstURL"`
	} `json:"RelatedTopics"`
}

// WebSearch performs a web search using DuckDuckGo Instant Answer API
func WebSearch(ctx context.Context, query string) (string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Build request URL
	searchURL := fmt.Sprintf("https://api.duckduckgo.com/?format=json&no_html=1&skip_disambig=1&q=%s",
		url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "zsh-ai-widget/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var ddgResp duckDuckGoResponse
	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	// Build result from available data
	var results []string

	// Direct answer (highest priority)
	if ddgResp.Answer != "" {
		results = append(results, "Answer: "+ddgResp.Answer)
	}

	// Definition
	if ddgResp.Definition != "" {
		results = append(results, "Definition: "+ddgResp.Definition)
	}

	// Abstract (summary from Wikipedia or other sources)
	if ddgResp.AbstractText != "" {
		source := ddgResp.AbstractSource
		if source == "" {
			source = "Source"
		}
		results = append(results, fmt.Sprintf("%s: %s", source, ddgResp.AbstractText))
	}

	// Related topics (limited to first 5)
	if len(ddgResp.RelatedTopics) > 0 {
		var topics []string
		for i, topic := range ddgResp.RelatedTopics {
			if i >= 5 {
				break
			}
			if topic.Text != "" {
				topics = append(topics, "- "+topic.Text)
			}
		}
		if len(topics) > 0 {
			results = append(results, "Related:\n"+strings.Join(topics, "\n"))
		}
	}

	if len(results) == 0 {
		return "No relevant results found for: " + query, nil
	}

	// Truncate to 4KB
	result := strings.Join(results, "\n\n")
	if len(result) > 4096 {
		result = result[:4096] + "..."
	}

	return result, nil
}
