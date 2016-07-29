package main

import (
	"flag"
	"net"
	"net/http"
	"os"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"log"
)

var (
	optFd uint

	optRegion string
	optBucket string
)

func init() {
	// file descriptor option for Circus
	flag.UintVar(&optFd, "fd", 0, "File descriptor to listen and serve.")
	flag.StringVar(&optRegion, "region", "", "AWS region")
	flag.StringVar(&optBucket, "bucket", "", "AWS S3 bucket name")

	// hiding goji -bind option.
	flag.Parse()
}

func main() {
	if optRegion == "" {
		log.Fatal("region is required.")
	}
	if optBucket == "" {
		log.Fatal("bucket is required.")
	}

	region := optRegion
	bucket := optBucket

	api, err := createAPI(region, bucket)
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

func createAPI(region, bucket string) (http.Handler, error) {
	api := NewAPI(region, bucket)

	apiMux := web.New()
	apiMux.Post("/api/bin", api.PostBin)
	apiMux.Get("/api/bin", api.GetBin)

	return apiMux, nil
}
