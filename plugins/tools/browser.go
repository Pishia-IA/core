package tools

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/Pishia-IA/core/config"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
	"github.com/sap-nocops/duckduckgogo/client"
)

type Browser struct {
}

func NewBrowser(config *config.Base) *Browser {
	return &Browser{}
}

const MAX_RESULTS_DUCK_DUCK_GO = 5

func (c *Browser) Run(params map[string]interface{}, userQuery string) (*ToolResponse, error) {
	var urlsToOpen []string

	// Handle direct URL requests
	if url, ok := params["url"].(string); ok {
		urlsToOpen = append(urlsToOpen, url)
	}

	// Handle search requests
	if searchQuery, ok := params["search"].(string); ok {
		ddg := client.NewDuckDuckGoSearchClient()
		searchResults, err := ddg.SearchLimited(searchQuery, MAX_RESULTS_DUCK_DUCK_GO)
		if err != nil {
			return nil, err
		}
		for _, result := range searchResults {
			log.Debugf("Found URL: %s", result.FormattedUrl)
			urlsToOpen = append(urlsToOpen, fmt.Sprintf("https://%s", result.FormattedUrl))
		}
	}

	// Channel to collect data from concurrent URL visits
	resultCh := make(chan string, len(urlsToOpen))

	// Visit each URL concurrently
	for _, url := range urlsToOpen {
		go func(url string) {
			if url == "" {
				resultCh <- ""
				return
			}
			page, err := c.visitURL(url)
			if err != nil {
				resultCh <- ""
				return
			}
			resultCh <- fmt.Sprintf("URL:%s\nDATA: %s\nNOTE: Ignore the cookie part", url, page)
		}(url)
	}

	// Collect results
	var prompts []string
	for i := 0; i < len(urlsToOpen); i++ {
		if prompt := <-resultCh; prompt != "" {
			prompts = append(prompts, prompt)
		}
	}

	return &ToolResponse{
		Success: true,
		Type:    "prompt",
		Prompts: prompts,
	}, nil
}

func (c *Browser) visitURL(url string) (string, error) {
	log.Debugf("Visiting URL: %s", url)
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var pageContent string
	if err := chromedp.Run(ctx, chromedp.Tasks{
		chromedp.Emulate(device.IPad),
		chromedp.Navigate(url),
		chromedp.Text("html", &pageContent),
	}); err != nil {
		return "", err
	}

	return pageContent, nil
}

func (c *Browser) Setup() error {
	return nil
}

func (c *Browser) Description() string {
	return "Browser is a tool that allows you to browse the web."
}

func (c *Browser) Parameters() map[string]*ToolParameter {
	return map[string]*ToolParameter{
		"search": {
			Type:        "string",
			Description: "The query to search on Browser.",
			Required:    false,
		},
		"url": {
			Type:        "string",
			Description: "The URL to open on Browser.",
			Required:    false,
		},
	}
}

func (c *Browser) UseCase() []string {
	return []string{
		"User is asking about current events or something that requires real-time information (news, weather, sports scores, etc.)",
		"User is asking about some term you are totally unfamiliar with (it might be new)",
		"User explicitly asks you to browse or provide links to references",
		"User is asking about an event that happened recently",
		"User send a URL and ask for a summary of the page",
	}
}