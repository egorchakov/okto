package main

import (
	"net/url"
	"reflect"
	"testing"
)

func TestFilter(t *testing.T) {
	f := NewFilter()
	parent, _ := url.Parse("http://abcd.com")
	children := []string{
		"http://abcd.com",
		"http://abcd.com/",
		"http://abcd.com/",
		"http://abcd.com/path",
		"http://abcd.com/path?param=1",
		"http://xyz.abcd.com/path?param=1",
		"http://notabcd.com",
	}

	expected := []string{
		"http://abcd.com",
		"http://abcd.com/path",
		"http://abcd.com/path?param=1",
	}

	expectedParsed := make([]*url.URL, len(expected))
	for i, s := range expected {
		p, _ := url.Parse(s)
		expectedParsed[i] = p
	}

	result := f.Filter(parent, children)

	if !reflect.DeepEqual(result, expectedParsed) {
		t.Logf("expected: %+v, got: %+v", expected, result)
		t.Fail()
	}
}
