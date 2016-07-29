package main

import (
	"log"
	"net/http"

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

	comment := r.FormValue("comment")
	if len(comment) > 140 {
		w.WriteHeader(400)
		w.Write([]byte("comment lenght lower equal 140"))
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

	meta := make(map[string]string, 1)
	meta["comment"] = comment
	data := &RNBinData{
		Sep:         sep,
		Name:        "",
		ContentType: contentType,
		Data:        img.Data,
		Metadata:    meta,
	}

	path, err := api.S3m.Store(data)
	if err != nil {
		panic(err)
	}

	w.Write([]byte(path))
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

	data, err := api.S3m.Get(name)
	if err != nil {
		panic(err)
	}

	w.Write(data)
}

func (api *API) GetMeta(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	if name == "" {
		w.WriteHeader(400)
		w.Write([]byte("require name parameter."))
		return
	}
	log.Printf("get file meta: %s", name)

	meta, err := api.S3m.GetMeta(name)
	if err != nil {
		panic(err)
	}
	log.Printf("get meta len: %d", len(meta))

	for k, v := range meta {
		w.Write([]byte(k))
		w.Write([]byte(":"))
		if v != nil {
			w.Write([]byte(*v))
		} else {
			w.Write([]byte("nil"))
		}
	}
}
