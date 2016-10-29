package main

import (
	"net/url"
	"os"
	"time"

	"github.com/Sirupsen/logrus"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var app = kingpin.New("okto", "Single-host crawler")

//conf contains parameters defined by CLI arsg/flags.
var conf struct {
	SeedURL *url.URL

	//Outbound request timeout
	Timeout time.Duration

	//Outbound request rate limit
	RateLimit uint64

	//Maximum number of concurrent connections
	MaxConn uint64

	//Output format and directory
	Format string
	Dir    string

	Debug bool
}

func parseArgs() {
	app.Arg("seed", "Seed URL to start crawling").
		Required().URLVar(&conf.SeedURL)

	app.Flag("timeout", "Timeout for individual URL fetch attempts").
		Default("10s").DurationVar(&conf.Timeout)

	app.Flag("rate", "Outgoing request rate limit (requests per second)").
		Default("0").Uint64Var(&conf.RateLimit)

	app.Flag("max-conn", "Maximum number of outgoing concurrent connections").
		Default("512").Uint64Var(&conf.MaxConn)

	app.Flag("debug", "Enable debug mode").
		BoolVar(&conf.Debug)

	app.Flag("dir", "Output directory").
		Default(".").ExistingDirVar(&conf.Dir)

	app.Flag("format", "Output format").
		Default(DotFormat).EnumVar(&conf.Format, Formats...)

	kingpin.MustParse(app.Parse(os.Args[1:]))
}

func main() {
	parseArgs()

	if conf.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	logrus.WithField("params", conf).Debug("Parsed parameters")

	crawler, err := NewCrawler(
		conf.SeedURL,
		conf.Timeout,
		conf.MaxConn,
		conf.RateLimit,
	)

	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize crawler")
	}

	result, err := crawler.Crawl()

	if err != nil {
		logrus.WithError(err).Fatal("Failed to crawl")
	}

	pp, err := NewPostProcessor(conf.Format)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to process results")
	}

	files, err := pp.Process(result).WriteToDir(conf.Dir)

	if err != nil {
		logrus.WithError(err).Fatal("Failed to write results")
	}

	logrus.WithField("files", files).Info("Done")
}
