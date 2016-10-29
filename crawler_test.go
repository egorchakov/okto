package main

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
)

type urlData map[string][]string

type mockFetcher struct{}

func (f *mockFetcher) Fetch(rawURL string) (*FetchResult, error) {
	var links []string
	var assets []string

	data := map[string]urlData{
		"http://abcd.com": urlData{
			"links": []string{
				"http://abcd.com/a",
				"http://external.com/a",
				"mailto://a@abcd.com",
			},
			"assets": []string{
				"http://abcd.com/img.png",
				"http://cdn.com/img.png",
			},
		},

		"http://abcd.com/a": urlData{
			"links": []string{
				"http://abcd.com/b",
			},
			"assets": []string{
				"http://abcd.com/img.png",
			},
		},
	}

	if v, ok := data[rawURL]; ok {
		links = v["links"]
		assets = v["assets"]
	}

	return &FetchResult{
		links:  links,
		assets: assets,
	}, nil
}

func TestCrawler(t *testing.T) {
	logrus.SetLevel(logrus.PanicLevel)

	seed, _ := url.Parse("http://abcd.com")
	crawler, _ := NewCrawler(seed, time.Duration(1), 1, 0)
	crawler.fetcher = &mockFetcher{}

	result, err := crawler.Crawl()
	if err != nil {
		t.Fail()
	}

	expectedLinks := map[string][]string{
		"http://abcd.com":   []string{"http://abcd.com/a"},
		"http://abcd.com/a": []string{"http://abcd.com/b"},
		"http://abcd.com/b": []string{},
	}

	expectedAssets := map[string][]string{
		"http://abcd.com":   []string{"http://abcd.com/img.png", "http://cdn.com/img.png"},
		"http://abcd.com/a": []string{"http://abcd.com/img.png"},
		"http://abcd.com/b": []string(nil),
	}

	if !(reflect.DeepEqual(result.Links, expectedLinks) && reflect.DeepEqual(result.Assets, expectedAssets)) {
		t.Fail()
	}
}
