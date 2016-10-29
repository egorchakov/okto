package main

import (
	"errors"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

type CrawlResult struct {
	Seed   *url.URL
	Links  map[string][]string
	Assets map[string][]string
}

// Semaphore signal.
type signal struct{}

type crawler struct {
	// Single-host crawler by design, so `seed` is a member rather than a function parameter.
	seed *url.URL

	// Semaphore (channel) to limit concurrent connections.
	connSem chan signal

	fetcher Fetcher
	filter  Filter

	// Keep track of visited links and data (child links, assets) separately.
	visited ConcurrentMap
	data    ConcurrentMap
}

// Whitelist extensions to avoid fetching heavy stuff (PDFs, executables, etc).
var extensionWhiteList = map[string]bool{
	"":      true,
	".html": true,
}

var errInvalidSeedURL = errors.New("Invalid seed URL")

func NewCrawler(seed *url.URL, timeout time.Duration, maxConn, rps uint64) (*crawler, error) {
	if !isValidSeed(seed) {
		return nil, errInvalidSeedURL
	}

	c := &crawler{
		connSem: make(chan signal, maxConn),
		fetcher: NewFetcher(timeout, rps),
		filter:  NewFilter(),
		visited: NewConcurrentMap(),
		data:    NewConcurrentMap(),
		seed:    seed,
	}

	return c, nil
}

func isValidSeed(seed *url.URL) bool {
	if seed.Scheme == "" || seed.Host == "" {
		return false
	}

	return true
}

// buildResult creates a structure interpretable by postprocessors.
func (c *crawler) buildResult() *CrawlResult {
	var result = &CrawlResult{
		Seed:   c.seed,
		Links:  make(map[string][]string),
		Assets: make(map[string][]string),
	}

	for link, data := range c.data.Map() {
		k := link.(string)

		switch data.(type) {
		case map[string][]string:
			v := data.(map[string][]string)
			result.Links[k] = v["links"]
			result.Assets[k] = v["assets"]

		case nil:
			result.Links[k] = nil
			result.Assets[k] = nil
		}
	}

	return result
}

// Crawl is the entry point that initiates crawling.
func (c *crawler) Crawl() (*CrawlResult, error) {
	var wg sync.WaitGroup

	logrus.WithField("seed", c.seed).Info("Starting")
	wg.Add(1)
	go c.crawl(c.seed, &wg)
	wg.Wait()

	logrus.WithField("url_cnt", len(c.data.Map())).Info("Finished crawling")

	return c.buildResult(), nil
}

// fetch calls the fetcher while limiting concurrent connections.
func (c *crawler) fetch(u *url.URL) (*FetchResult, error) {
	defer func() { <-c.connSem }()
	c.connSem <- signal{}

	return c.fetcher.Fetch(u.String())
}

// crawl is the crawling worker that visits URLs, records links and assets, and spawns goroutines for child ULRs (under certain conditions).
func (c *crawler) crawl(u *url.URL, wg *sync.WaitGroup) {
	defer wg.Done()

	// No-op if we've already visited this URL.
	if unseen := c.visited.SetIfAbsent(u.String(), struct{}{}); !unseen {
		return
	}

	// Check if the URL should be crawled.
	if !c.shouldCrawl(u) {
		c.data.Set(u.String(), nil)
		logrus.WithField("url", u).Debug("Skipping")
		return
	}

	// If fetching fails, log and move on.
	fetchResult, err := c.fetch(u)
	if err != nil {
		logrus.WithField("url", u).WithError(err).Error("Failed to fetch")
		return
	}

	// Filter child URLs.
	urls := c.filter.Filter(u, fetchResult.links)
	logrus.WithField("url", u).WithField("children", urls).Debug("Filtered")

	// Convert child URLs to strings and spawn goroutines.
	links := make([]string, len(urls))
	for i, u := range urls {

		defer func(u *url.URL) {
			wg.Add(1)
			go c.crawl(u, wg)
		}(u)

		links[i] = u.String()
	}

	// Finally, record links and assets for this URL.
	c.data.Set(u.String(), map[string][]string{
		"links":  links,
		"assets": fetchResult.assets,
	})

	logrus.WithField("url", u).Info("Crawled")
}

func (c *crawler) shouldCrawl(u *url.URL) bool {
	if u.Path != "" && !extensionWhiteList[filepath.Ext(u.Path)] {
		return false
	}

	return true
}
