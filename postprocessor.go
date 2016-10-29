package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	graphviz "github.com/awalterschulze/gographviz"
)

const (
	JSONFormat = "json"
	DotFormat  = "dot"
)

var (
	Formats               = []string{JSONFormat, DotFormat}
	errFormatNotSupported = errors.New("Format not supported")
)

type (
	PostProcessorResult interface {
		WriteToDir(string) ([]string, error)
	}

	PostProcessor interface {
		Process(*CrawlResult) PostProcessorResult
	}
)

func NewPostProcessor(format string) (PostProcessor, error) {
	switch format {
	case JSONFormat:
		return NewJSONPostProcessor(), nil
	case DotFormat:
		return NewDotPostProcessor(), nil
	default:
		return nil, errFormatNotSupported
	}
}

type (
	jsonPostProcessor struct{}

	jsonPostProcessorResult struct {
		Seed *url.URL
		Data []byte
	}
)

func NewJSONPostProcessor() PostProcessor {
	return &jsonPostProcessor{}
}

func (p *jsonPostProcessor) Process(result *CrawlResult) PostProcessorResult {
	data := map[string]map[string][]string{
		"links":  result.Links,
		"assets": result.Assets,
	}

	dataJSON, _ := json.Marshal(data)

	return &jsonPostProcessorResult{
		Seed: result.Seed,
		Data: dataJSON,
	}
}

func (r *jsonPostProcessorResult) WriteToDir(dir string) ([]string, error) {
	var err error

	file := filepath.Join(dir, fmt.Sprintf("%s.json", r.Seed.Host))

	if err = ioutil.WriteFile(file, r.Data, 0644); err != nil {
		return nil, err
	}

	return []string{file}, nil
}

type (
	dotPostProcessor struct{}

	dotPostProcessorResult struct {
		Seed       *url.URL
		LinkGraph  string
		AssetGraph string
	}
)

func NewDotPostProcessor() PostProcessor {
	return &dotPostProcessor{}
}

func (r *dotPostProcessorResult) WriteToDir(dir string) ([]string, error) {
	var err error

	linkFile := filepath.Join(dir, fmt.Sprintf("%s_links.dot", r.Seed.Host))
	assetFile := filepath.Join(dir, fmt.Sprintf("%s_assets.dot", r.Seed.Host))

	if err = ioutil.WriteFile(linkFile, []byte(r.LinkGraph), 0644); err != nil {
		return nil, err
	}

	if err = ioutil.WriteFile(assetFile, []byte(r.AssetGraph), 0644); err != nil {
		return nil, err
	}

	return []string{linkFile, assetFile}, nil
}

func (p *dotPostProcessor) Process(result *CrawlResult) PostProcessorResult {
	return &dotPostProcessorResult{
		Seed:       result.Seed,
		LinkGraph:  p.buildLinkGraph(result.Links).String(),
		AssetGraph: p.buildAssetGraph(result.Assets).String(),
	}
}

func (p *dotPostProcessor) buildLinkGraph(links map[string][]string) *graphviz.Graph {
	var childPath, parentPath string
	var graphName = "links"

	g := graphviz.NewGraph()
	g.SetName("links")
	g.SetDir(true)

	for parent, children := range links {
		parent, err := url.Parse(parent)

		if err != nil {
			logrus.WithField("url", parent).WithError(err).Error("Failed to parse")
			continue
		}

		parentPath = p.normalize(parent)

		g.AddNode(graphName, parentPath, nil)

		for _, child := range children {
			child, err := url.Parse(child)

			if err != nil {
				logrus.WithField("url", child).WithError(err).Error("Failed to parse")
				continue
			}

			childPath = p.normalize(child)

			g.AddNode(graphName, childPath, nil)
			g.AddEdge(parentPath, childPath, true, nil)
		}
	}

	return g
}

func (p *dotPostProcessor) buildAssetGraph(assets map[string][]string) *graphviz.Graph {
	var childPath, parentPath string
	var graphName = "assets"

	childAttrs := map[string]string{"color": "red"}

	g := graphviz.NewGraph()
	g.SetName(graphName)
	g.SetDir(true)

	for parent, children := range assets {

		if len(children) == 0 {
			continue
		}

		parent, err := url.Parse(parent)

		if err != nil {
			logrus.WithField("url", parent).WithError(err).Error("Failed to parse")
			continue
		}

		parentPath = p.normalize(parent)
		g.AddNode(graphName, parentPath, nil)

		for _, child := range children {
			child, err := url.Parse(child)

			if err != nil {
				logrus.WithField("url", child).WithError(err).Error("Failed to parse")
				continue
			}

			childPath = p.escape(child.String())

			g.AddNode(graphName, childPath, childAttrs)
			g.AddEdge(parentPath, childPath, true, nil)
		}
	}
	return g
}

func (p *dotPostProcessor) normalize(u *url.URL) string {
	if !(u.Path == "" || u.Path == "/") {
		return p.escape(u.Path)
	}

	return p.escape(u.String())
}

func (p *dotPostProcessor) escape(s string) string {
	return fmt.Sprintf("\"%s\"", s)
}
