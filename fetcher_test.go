package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestFetcherProcessBody(t *testing.T) {
	f := &fetcher{}

	var htmlStr = `
	<head profile="http://head.profile">
	<a href="http://a.href"> </a>
	<iframe src="http://iframe.src"> </iframe>
	<q cite="http://q.cite"></q>

	<link href="http://link.href">
	<img src="http://img.src">
	<script src="http://script.src">
	`
	expectedLinks := []string{
		"http://head.profile",
		"http://a.href",
		"http://iframe.src",
		"http://q.cite",
	}

	expectedAssets := []string{
		"http://link.href",
		"http://img.src",
		"http://script.src",
	}

	links, assets := f.processBody(strings.NewReader(htmlStr))

	if !reflect.DeepEqual(links, expectedLinks) {
		t.Logf("expected: %+v, got: %+v", expectedLinks, links)
		t.Fail()
	}

	if !reflect.DeepEqual(assets, expectedAssets) {
		t.Logf("expected: %+v, got: %+v", expectedAssets, assets)
		t.Fail()
	}
}
