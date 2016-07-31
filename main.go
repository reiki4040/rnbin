package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"log"
)

var (
	version   string
	hash      string
	goversion string

	optVersion bool

	optFd uint

	optRegion  string
	optBuckets string
)

func init() {
	flag.BoolVar(&optVersion, "v", false, "show version")
	flag.BoolVar(&optVersion, "version", false, "show version")

	// file descriptor option for Circus
	flag.UintVar(&optFd, "fd", 0, "File descriptor to listen and serve.")
	flag.StringVar(&optRegion, "region", "", "AWS region")
	flag.StringVar(&optBuckets, "buckets", "", "AWS S3 bucket names separated with comma")

	// hiding goji -bind option.
	flag.Parse()
}

func showVersion() {
	fmt.Printf("RNBin %s %s %s\n", version, hash, goversion)
}

func main() {
	if optVersion {
		showVersion()
		return
	}

	if optRegion == "" {
		log.Fatal("region is required.")
	}
	if optBuckets == "" {
		log.Fatal("bucket is required.")
	}
	buckets := strings.Split(optBuckets, ",")
	for _, b := range buckets {
		if b == "" {
			log.Fatal("buckets includes empty name")
		}
	}

	region := optRegion

	api, err := createAPI(region, buckets)
	if err != nil {
		panic(err)
	}

	rootMux := goji.DefaultMux
	rootMux.Handle("/api/*", api)

	if optFd != 0 {
		l, err := net.FileListener(os.NewFile(uintptr(optFd), ""))
		if err != nil {
			panic(err)
		}

		goji.ServeListener(l)
	} else {
		// if not specified fd, then goji default(:8000 or -bind arg)
		goji.Serve()
	}
}

func createAPI(region string, bucket []string) (http.Handler, error) {
	api := NewAPI(region, bucket)

	apiMux := web.New()
	apiMux.Post("/api/bin", api.PostBin)
	apiMux.Get("/api/bin", api.GetBin)
	apiMux.Get("/api/meta", api.GetMeta)

	return apiMux, nil
}
