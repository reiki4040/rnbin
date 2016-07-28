package main

import (
	"io"
	//"os"

	//"compress/gzip"
	"crypto/sha256"
	"encoding/hex"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

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

/*
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
*/

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

func Sha256(bytes []byte) string {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hex.EncodeToString(hasher.Sum(nil))
}
