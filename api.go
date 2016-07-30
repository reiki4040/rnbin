package main

import (
	"encoding/json"
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
		responseBadRequest(w, "sep is required")
		return
	}

	comment := r.FormValue("comment")
	if len(comment) > 140 {
		responseBadRequest(w, "comment lenght lower equal 140")
		return
	}

	contentType := img.ContentType
	if contentType == "" {
		responseBadRequest(w, "content type is required")
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

	responseOK(w, &PostBinResp{Path: path})
}

type PostBinResp struct {
	Path string `json:"path"`
}

func (api *API) GetBin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	if name == "" {
		responseBadRequest(w, "require name parameter.")
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
		responseBadRequest(w, "require name parameter.")
		return
	}
	log.Printf("get file meta: %s", name)

	meta, err := api.S3m.GetMeta(name)
	if err != nil {
		panic(err)
	}
	log.Printf("get meta len: %d", len(meta))

	resp := ConvertMeta(meta)

	responseOK(w, resp)
}

func responseOK(w http.ResponseWriter, resp interface{}) {
	responseJson(w, http.StatusOK, resp)
}

func responseBadRequest(w http.ResponseWriter, msg string) {
	responseError(w, http.StatusBadRequest, msg)
}

func responseError(w http.ResponseWriter, status int, msg string) {
	responseJson(w, status, NewErrResp(msg))
}

func responseJson(w http.ResponseWriter, status int, item interface{}) {
	j, err := json.Marshal(item)
	if err != nil {
		// TODO internal server error
		panic(err)
	}

	writeResponse(w, status, j)
}

func writeResponse(w http.ResponseWriter, status int, body []byte) {
	w.WriteHeader(status)
	w.Write(body)
}

func ConvertMeta(m map[string]*string) *Meta {
	meta := &Meta{}

	key := m["key"]
	if key != nil {
		meta.Key = *key
	}

	createBy := m["create_by"]
	if createBy != nil {
		meta.CreateBy = *createBy
	}

	comment := m["comment"]
	if comment != nil {
		meta.Comment = *comment
	}

	return meta
}

type MetaResponse struct {
	Metadata Meta `json:"metadata"`
}

type Meta struct {
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	CreateBy    string `json:"create_by"`
	Comment     string `json:"comment"`
}

func NewErrResp(msg string) *ErrorResponse {
	return &ErrorResponse{Msg: msg}
}

type ErrorResponse struct {
	Msg string `json:"error_message"`
}
