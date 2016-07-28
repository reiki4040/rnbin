package main

import (
	"bytes"
	"flag"
	"net"
	"net/http"
	"os"

	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"

	"github.com/olahol/go-imageupload"

	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	apiMux.Handle("/api/upload", api.UploadFile)
	apiMux.Handle("/api/download", api.DownloadFile)

	return apiMux, nil
}

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

func s3updown() {
	s3 := NewS3("ap-northeast-1")

	// upload
	r, name, err := s3.Upload("always-ice", "gopher.png", "image/png", false)
	if err != nil {
		log.Fatalf("error upload to s3: %s", err.Error())
	} else {
		log.Printf("upload complete to %s", r.Location)
	}

	// download
	file, err := os.Create("download_file")
	if err != nil {
		log.Fatal("Failed to create file", err)
	}
	defer file.Close()

	numBytes, err := s3.Download("always-ice", name, file)
	if err != nil {
		log.Fatal("failed to download:", err)
	}
	log.Println("uploaded file and download it testing complated.")
	log.Printf("get file: %s, bytes: %d\n", name, numBytes)
}

func Sha256FromFile(filepath string) (string, error) {
	_, err := os.Open(filepath)
	if err != nil {
		return "", err
	}

	s, err := ioutil.ReadFile(filepath)
	if err != nil {
		return "", err
	}

	return Sha256(s), nil
}

func Sha256(bytes []byte) string {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hex.EncodeToString(hasher.Sum(nil))
}

func NewS3(region string) *S3 {
	session := session.New(&aws.Config{Region: aws.String(region)})
	uploader := s3manager.NewUploader(session)
	downloader := s3manager.NewDownloader(session)

	return &S3{
		Uploader:   uploader,
		Downloader: downloader,
	}
}

type S3 struct {
	Uploader   *s3manager.Uploader
	Downloader *s3manager.Downloader
}

func (s3 *S3) UploadFromReader(bucket, name, contentType string, r io.Reader) (*s3manager.UploadOutput, error) {
	// UploadInput
	// https://github.com/aws/aws-sdk-go/blob/master/service/s3/s3manager/upload.go#L99
	result, err := s3.Uploader.Upload(&s3manager.UploadInput{
		Body:        r,
		Bucket:      aws.String(bucket),
		Key:         aws.String(name),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return nil, err
	} else {
		return result, nil
	}
}
func (s3 *S3) Upload(bucket, filepath, contentType string, compGzip bool) (*s3manager.UploadOutput, string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, "", err
	}

	var reader io.ReadCloser
	var writer io.WriteCloser

	reader = file
	if compGzip {
		// Not required, but you could zip the file before uploading it
		// using io.Pipe read/writer to stream gzip'd file contents.
		reader, writer = io.Pipe()
		go func() {
			gw := gzip.NewWriter(writer)
			io.Copy(gw, file)

			file.Close()
			gw.Close()
			writer.Close()
		}()
	}

	hash, err := Sha256FromFile(filepath)
	if err != nil {
		return nil, "", err
	}

	// UploadInput
	// https://github.com/aws/aws-sdk-go/blob/master/service/s3/s3manager/upload.go#L99
	result, err := s3.Uploader.Upload(&s3manager.UploadInput{
		Body:        reader,
		Bucket:      aws.String(bucket),
		Key:         aws.String(hash),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return nil, "", err
	} else {
		return result, hash, nil
	}
}

func (s3m *S3) Download(bucket, key string, w io.WriterAt) (int64, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	numBytes, err := s3m.Downloader.Download(w, input)
	if err != nil {
		return 0, err
	}

	return numBytes, nil
}
