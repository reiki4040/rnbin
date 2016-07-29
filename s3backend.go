package main

import (
	"bytes"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type RNBinData struct {
	Sep         string
	Name        string
	ContentType string
	Data        []byte
}

func (d *RNBinData) GenDistHead() string {
	hash := Sha256(d.Data)
	return hash[:6] + "-" + d.Sep + "/" + hash
}

type Backend interface {
	Store(*RNBinData) (map[string]string, error)
	StoreWithReader(path, contentType string, reader io.Reader) (map[string]string, error)
	Get(name string) (*RNBinData, error)
	GetToWriteAt(path, w io.WriterAt) (int64, error)
}

func NewS3Backend(region, bucket string) *S3Backend {
	session := session.New(&aws.Config{Region: aws.String(region)})
	uploader := s3manager.NewUploader(session)
	downloader := s3manager.NewDownloader(session)

	return &S3Backend{
		BucketName: bucket,
		Uploader:   uploader,
		Downloader: downloader,
	}
}

type S3Backend struct {
	BucketName string
	Uploader   *s3manager.Uploader
	Downloader *s3manager.Downloader
}

func (s3 *S3Backend) Store(data *RNBinData) (map[string]string, error) {

	r := bytes.NewReader(data.Data)
	return s3.StoreWithReader(data.GenDistHead(), data.Name, data.ContentType, r)
}

func (s3 *S3Backend) StoreWithReader(path, name, contentType string, reader io.Reader) (map[string]string, error) {
	// UploadInput
	// https://github.com/aws/aws-sdk-go/blob/master/service/s3/s3manager/upload.go#L99
	_, err := s3.Uploader.Upload(&s3manager.UploadInput{
		Body:        reader,
		Bucket:      aws.String(s3.BucketName),
		Key:         aws.String(path),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return nil, err
	} else {
		m := make(map[string]string, 1)
		m["name"] = path
		return m, nil
	}
}

func (s3 *S3Backend) Get(path string) (*RNBinData, error) {
	buf := new(aws.WriteAtBuffer)
	_, err := s3.GetToWriteAt(path, buf)
	if err != nil {
		return nil, err
	}

	data := &RNBinData{
		Data: buf.Bytes(),
	}

	return data, nil
}

func (s3m *S3Backend) GetToWriteAt(path string, w io.WriterAt) (int64, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s3m.BucketName),
		Key:    aws.String(path),
	}

	numBytes, err := s3m.Downloader.Download(w, input)
	if err != nil {
		return 0, err
	}

	return numBytes, nil
}
