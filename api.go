package main

import (
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/olahol/go-imageupload"
)

func NewAPI(region, bucket string) *API {
	return &API{
		BucketName: bucket,
		S3m:        NewS3Backend(region, bucket),
	}
}

type API struct {
	BucketName string
	S3m        *S3Backend
}

func (api *API) PostBin(w http.ResponseWriter, r *http.Request) {
	// get from 'file' request parameter
	img, err := imageupload.Process(r, "file")
	if err != nil {
		panic(err)
	}

	sep := r.FormValue("sep")
	if sep == "" {
		w.WriteHeader(400)
		w.Write([]byte("sep is required"))
		return
	}

	contentType := img.ContentType
	if contentType == "" {
		w.WriteHeader(400)
		w.Write([]byte("content type is required"))
		return
	}

	log.Printf("uploaded %s", contentType)
	log.Printf("uploaded %d bytes data", r.ContentLength)

	data := &RNBinData{
		Sep:         sep,
		Name:        "",
		ContentType: contentType,
		Data:        img.Data,
	}

	m, err := api.S3m.Store(data)
	if err != nil {
		panic(err)
	}

	w.Write([]byte(m["name"]))
}

func (api *API) GetBin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	if name == "" {
		w.WriteHeader(400)
		w.Write([]byte("require name parameter."))
		return
	}
	log.Printf("get file: %s", name)

	wab := new(aws.WriteAtBuffer)
	_, err := api.S3m.GetToWriteAt(name, wab)
	if err != nil {
		panic(err)
	}

	w.Write(wab.Bytes())
}
