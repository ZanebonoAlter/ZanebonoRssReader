package feeds

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

type RSSParser struct {
	client *http.Client
}

func NewRSSParser() *RSSParser {
	return &RSSParser{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type ParsedFeed struct {
	Title       string
	Description string
	Link        string
	Image       string
	Language    string
	Entries     []ParsedEntry
}

type ParsedEntry struct {
	Title       string
	Link        string
	Description string
	Content     string
	PubDate     *time.Time
	Author      string
	Tags        []string
	ImageURL    string
}

func (p *RSSParser) ParseFeedURL(feedURL string) (*ParsedFeed, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	return p.convertGofeedToParsed(feed), nil
}

func (p *RSSParser) ParseFeedFromString(xmlContent string) (*ParsedFeed, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseString(xmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	return p.convertGofeedToParsed(feed), nil
}

func (p *RSSParser) convertGofeedToParsed(feed *gofeed.Feed) *ParsedFeed {
	parsed := &ParsedFeed{
		Title:       feed.Title,
		Description: feed.Description,
		Link:        feed.Link,
		Image:       extractFeedImage(feed),
		Language:    feed.Language,
		Entries:     make([]ParsedEntry, 0, len(feed.Items)),
	}

	for _, item := range feed.Items {
		entry := ParsedEntry{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Content:     extractContent(item),
			PubDate:     parseDate(item),
			Author:      extractAuthor(item),
			Tags:        extractFeedTags(item),
			ImageURL:    extractItemImage(item),
		}

		if entry.Title == "" {
			entry.Title = "No title"
		}

		parsed.Entries = append(parsed.Entries, entry)
	}

	return parsed
}

func extractFeedImage(feed *gofeed.Feed) string {
	if feed.Image != nil {
		return feed.Image.URL
	}
	return ""
}

func extractContent(item *gofeed.Item) string {
	if item.Content != "" {
		return item.Content
	}
	return item.Description
}

func parseDate(item *gofeed.Item) *time.Time {
	cstZone := time.FixedZone("CST", 8*3600)

	if item.PublishedParsed != nil {
		cstTime := item.PublishedParsed.In(cstZone)
		return &cstTime
	}
	if item.UpdatedParsed != nil {
		cstTime := item.UpdatedParsed.In(cstZone)
		return &cstTime
	}
	return nil
}

func extractAuthor(item *gofeed.Item) string {
	if item.Author != nil {
		return item.Author.Name
	}
	return ""
}

func extractFeedTags(item *gofeed.Item) []string {
	tags := make([]string, 0)
	for _, tag := range item.Categories {
		tags = append(tags, tag)
	}
	return tags
}

func extractItemImage(item *gofeed.Item) string {
	if item.Enclosures != nil && len(item.Enclosures) > 0 {
		for _, enc := range item.Enclosures {
			if strings.HasPrefix(enc.Type, "image/") {
				return enc.URL
			}
		}
	}
	return ""
}

func (p *RSSParser) FetchFaviconURL(feedURL string) string {
	parsedURL, err := url.Parse(feedURL)
	if err != nil {
		return "rss"
	}

	scheme := "https"
	if parsedURL.Scheme == "http" {
		scheme = "http"
	}

	return fmt.Sprintf("%s://%s/favicon.ico", scheme, parsedURL.Host)
}

func (p *RSSParser) ValidateFeedURL(feedURL string) bool {
	if feedURL == "" {
		return false
	}

	_, err := url.Parse(feedURL)
	if err != nil {
		return false
	}

	lowercaseURL := strings.ToLower(feedURL)
	feedPatterns := []string{"/feed", "/rss", "/atom", "rss.xml", "feed.xml", "atom.xml", ".rss", ".atom"}

	for _, pattern := range feedPatterns {
		if strings.Contains(lowercaseURL, pattern) {
			return true
		}
	}

	return true
}

func (p *RSSParser) FetchFeedMetadata(feedURL string) (title, description string, err error) {
	parsed, err := p.ParseFeedURL(feedURL)
	if err != nil {
		return "", "", err
	}

	return parsed.Title, parsed.Description, nil
}
