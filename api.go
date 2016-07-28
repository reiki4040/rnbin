package main

import (
	"bytes"
	"net/http"
	//"os"
	//"github.com/zenazn/goji"
	//"github.com/zenazn/goji/web"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/olahol/go-imageupload"
)

func NewAPI(region, bucket string) *API {
	return &API{
		BucketName: bucket,
		S3m:        NewS3(region),
	}
}

type API struct {
	BucketName string
	S3m        *S3
}

func (api *API) UploadFile(w http.ResponseWriter, r *http.Request) {
	// get from 'file' request parameter
	img, err := imageupload.Process(r, "file")
	if err != nil {
		panic(err)
	}

	log.Printf("uploaded %d bytes file", len(img.Data))
	hash := Sha256(img.Data)
	if err != nil {
		return
	}
	log.Printf("hash: %s", hash)

	reader := bytes.NewReader(img.Data)
	_, err = api.S3m.UploadFromReader(api.BucketName, hash, "image/png", reader)
	if err != nil {
		panic(err)
	}

	w.Write([]byte(hash))
}

func (api *API) DownloadFile(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.Form.Get("name")
	if name == "" {
		w.WriteHeader(400)
		w.Write([]byte("require name parameter."))
		return
	}
	log.Printf("get file: %s", name)

	wab := new(aws.WriteAtBuffer)
	_, err := api.S3m.Download(api.BucketName, name, wab)
	if err != nil {
		panic(err)
	}

	w.Write(wab.Bytes())
}
