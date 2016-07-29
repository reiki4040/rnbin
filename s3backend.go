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
	Metadata    map[string]string
}

func (d *RNBinData) GenDistHead() string {
	hash := Sha256(d.Data)
	return hash[:6] + "-" + d.Sep + "/" + hash
}

func (d *RNBinData) convmeta() map[string]*string {
	m := make(map[string]*string)
	for k, v := range d.Metadata {
		m[k] = &v
	}

	return m
}

func NewS3Backend(region, bucket string) *S3Backend {
	session := session.New(&aws.Config{Region: aws.String(region)})
	s3srv := s3.New(session)
	uploader := s3manager.NewUploader(session)
	downloader := s3manager.NewDownloader(session)

	return &S3Backend{
		BucketName: bucket,
		Uploader:   uploader,
		Downloader: downloader,
		S3Srv:      s3srv,
	}
}

type S3Backend struct {
	BucketName string
	Uploader   *s3manager.Uploader
	Downloader *s3manager.Downloader
	S3Srv      *s3.S3
}

func (s3 *S3Backend) Store(data *RNBinData) (string, error) {
	r := bytes.NewReader(data.Data)
	return s3.StoreWithReader(data.GenDistHead(), data.Name, data.ContentType, r, data.convmeta())
}

func (s3 *S3Backend) StoreWithReader(path, name, contentType string, reader io.Reader, meta map[string]*string) (string, error) {
	// UploadInput
	// https://github.com/aws/aws-sdk-go/blob/master/service/s3/s3manager/upload.go#L99

	_, err := s3.Uploader.Upload(&s3manager.UploadInput{
		Body:        reader,
		Bucket:      aws.String(s3.BucketName),
		Key:         aws.String(path),
		ContentType: aws.String(contentType),
		Metadata:    meta,
	})

	if err != nil {
		return "", err
	} else {
		return path, nil
	}
}

func (s3 *S3Backend) Get(path string) ([]byte, error) {
	buf := new(aws.WriteAtBuffer)
	_, err := s3.GetToWriteAt(path, buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
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

func (s3m *S3Backend) GetMeta(path string) (map[string]*string, error) {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(s3m.BucketName),
		Key:    aws.String(path),
	}

	resp, err := s3m.S3Srv.HeadObject(params)
	if err != nil {
		return nil, err
	}

	return resp.Metadata, nil
}
