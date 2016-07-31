package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/olahol/go-imageupload"

	s3b "github.com/reiki4040/rnbin/s3backend"
)

const (
	DEFAULT_CREATED_BY = "RNBin"

	TIME_FORMAT = "2006-01-02T03:04:05"
)

func NewAPI(region string, buckets []string) *API {
	return &API{
		S3m: s3b.NewS3Backend(region, buckets),
	}
}

type API struct {
	S3m *s3b.S3Backend
}

func (api *API) PostBin(w http.ResponseWriter, r *http.Request) {
	// get from 'file' request parameter
	img, err := imageupload.Process(r, "file")
	if err != nil {
		panic(err)
	}

	name := r.FormValue("name")
	if name == "" {
		responseBadRequest(w, "name is required")
		return
	}

	sep := r.FormValue("sep")
	if sep == "" {
		responseBadRequest(w, "sep is required")
		return
	}

	createdBy := r.FormValue("createdBy")
	if createdBy == "" {
		createdBy = DEFAULT_CREATED_BY
	}

	if len(createdBy) > 100 {
		responseBadRequest(w, "created by lenght lower equal 100")
		return
	}

	contentType := img.ContentType
	if contentType == "" {
		responseBadRequest(w, "content type is required")
		return
	}

	log.Printf("uploaded %s", contentType)
	log.Printf("uploaded %d bytes data", r.ContentLength)

	data := &s3b.RNBinData{
		OriginName:  name,
		ContentType: contentType,
		Data:        img.Data,
		CreatedBy:   createdBy,
		Sep:         sep,
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
	key := r.FormValue("key")
	if key == "" {
		responseBadRequest(w, "require key parameter.")
		return
	}
	log.Printf("get file: %s", key)

	data, err := api.S3m.Get(key)
	if err != nil {
		panic(err)
	}

	w.Write(data)
}

func (api *API) GetMeta(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	key := r.FormValue("key")
	if key == "" {
		responseBadRequest(w, "require key parameter.")
		return
	}
	log.Printf("get file meta: %s", key)

	meta, err := api.S3m.GetMeta(key)
	if err != nil {
		switch err {
		case s3b.ERR_OUTOFBOUNDS_BUCKET_POSITION:
			responseError(w, http.StatusInternalServerError, "internal server error")
			return
		case s3b.ERR_FILE_NOT_FOUND:
			responseError(w, http.StatusNotFound, "file not found")
			return
		default:
			panic(err)
		}
	}

	resp := ConvertS3Resp(meta)
	resp.Key = key

	responseOK(w, resp)
}

// caution: s3 metadata key name first char is Upper
func ConvertS3Resp(m *s3b.Meta) *MetaResponse {
	lm := TtoA(&m.LastModified)

	meta := &RespMeta{
		ContentType:   m.ContentType,
		ContentLength: m.ContentLength,
		LastModified:  lm,

		OriginName: m.OriginName,
		Sep:        m.Sep,
		CreatedBy:  m.CreatedBy,
	}

	return &MetaResponse{Metadata: meta}
}

type MetaResponse struct {
	Key      string    `json:"key"`
	Metadata *RespMeta `json:"metadata"`
}

type RespMeta struct {
	ContentType   string `json:"content_type"`
	ContentLength int64  `json:"content_length"`
	LastModified  string `json:"last_modified"`

	OriginName string `json:"origin_name"`
	Sep        string `json:"sep"`
	CreatedBy  string `json:"created_by"`
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

func NewErrResp(msg string) *ErrorResponse {
	return &ErrorResponse{Msg: msg}
}

type ErrorResponse struct {
	Msg string `json:"error_message"`
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

func TtoA(t *time.Time) string {
	return t.Format(TIME_FORMAT)
}
