package s3backend

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"crypto/sha256"
	"encoding/hex"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	SEP_S3_DIR   = "/"
	SEP_SUFFIX   = "-"
	SEP_POSITION = "@"
)

var (
	hexMap = map[string]int{
		"0": 0,
		"1": 1,
		"2": 2,
		"3": 3,
		"4": 4,
		"5": 5,
		"6": 6,
		"7": 7,
		"8": 8,
		"9": 9,
		"a": 10,
		"b": 11,
		"c": 12,
		"d": 13,
		"e": 14,
		"f": 15,
	}

	ERR_OUTOFBOUNDS_BUCKET_POSITION = errors.New("out bounds bucket position")
	ERR_FILE_NOT_FOUND              = errors.New("file not found")
)

type RNBinData struct {
	Sep         string
	Name        string
	ContentType string
	Data        []byte
	Metadata    map[string]string
}

func (d *RNBinData) convmeta() map[string]*string {
	m := make(map[string]*string)
	for k, v := range d.Metadata {
		m[k] = &v
	}

	return m
}

func NewS3Backend(region string, buckets []string) *S3Backend {
	session := session.New(&aws.Config{Region: aws.String(region)})
	s3srv := s3.New(session)
	uploader := s3manager.NewUploader(session)
	downloader := s3manager.NewDownloader(session)

	return &S3Backend{
		BucketNames:  buckets,
		Uploader:     uploader,
		Downloader:   downloader,
		S3Srv:        s3srv,
		Distribution: len(buckets),
	}
}

type S3Backend struct {
	BucketNames  []string
	Uploader     *s3manager.Uploader
	Downloader   *s3manager.Downloader
	S3Srv        *s3.S3
	Distribution int
}

func (s3 *S3Backend) GetBucketName(pos int) (string, error) {
	if len(s3.BucketNames) <= pos {
		return "", ERR_OUTOFBOUNDS_BUCKET_POSITION
	}

	return s3.BucketNames[pos], nil
}

func (s3 *S3Backend) Store(data *RNBinData) (string, error) {
	r := bytes.NewReader(data.Data)
	path, pos := GenPathAndDistPosition(data.Data, data.Sep, s3.Distribution)
	return s3.StoreWithReader(pos, path, data.Name, data.ContentType, r, data.convmeta())
}

func (s3 *S3Backend) StoreWithReader(pos int, path, name, contentType string, reader io.Reader, meta map[string]*string) (string, error) {
	// UploadInput
	// https://github.com/aws/aws-sdk-go/blob/master/service/s3/s3manager/upload.go#L99

	bucket, err := s3.GetBucketName(pos)
	if err != nil {
		return "", err
	}

	_, err = s3.Uploader.Upload(&s3manager.UploadInput{
		Body:        reader,
		Bucket:      aws.String(bucket),
		Key:         aws.String(path),
		ContentType: aws.String(contentType),
		Metadata:    meta,
	})

	if err != nil {
		return "", err
	} else {
		return path + SEP_POSITION + strconv.Itoa(pos), nil
	}
}

func (s3 *S3Backend) Get(key string) ([]byte, error) {
	path, pos, err := resolvePosition(key)
	if err != nil {
		return nil, err
	}

	buf := new(aws.WriteAtBuffer)
	_, err = s3.GetToWriteAt(pos, path, buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s3m *S3Backend) GetToWriteAt(pos int, path string, w io.WriterAt) (int64, error) {
	bucket, err := s3m.GetBucketName(pos)
	if err != nil {
		return -1, err
	}

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}

	numBytes, err := s3m.Downloader.Download(w, input)
	if err != nil {
		return 0, err
	}

	return numBytes, nil
}

func (s3m *S3Backend) GetMeta(key string) (map[string]*string, error) {
	path, pos, err := resolvePosition(key)
	if err != nil {
		return nil, err
	}

	bucket, err := s3m.GetBucketName(pos)
	if err != nil {
		return nil, err
	}

	params := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}

	resp, err := s3m.S3Srv.HeadObject(params)
	if err != nil {
		return nil, resolveS3Error(err)
	}

	return resp.Metadata, nil
}

func (s3m *S3Backend) GetObject(key string) ([]byte, map[string]*string, error) {
	path, pos, err := resolvePosition(key)
	if err != nil {
		return nil, nil, err
	}

	r, meta, err := s3m.GetObjectWithReadCloser(pos, path)
	if err != nil {
		return nil, nil, err
	}
	defer r.Close()

	sc := bufio.NewScanner(r)
	data := sc.Bytes()

	return data, meta, nil
}

func (s3m *S3Backend) GetObjectWithReadCloser(pos int, path string) (io.ReadCloser, map[string]*string, error) {
	bucket, err := s3m.GetBucketName(pos)
	if err != nil {
		return nil, nil, err
	}

	params := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(path),
	}

	resp, err := s3m.S3Srv.GetObject(params)
	if err != nil {
		return nil, nil, err
	}

	return resp.Body, resp.Metadata, nil
}

func Sha256(bytes []byte) string {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hex.EncodeToString(hasher.Sum(nil))
}

func GenPathAndDistPosition(data []byte, sep string, dist int) (string, int) {
	hash := Sha256(data)
	suffix := strconv.FormatInt(time.Now().Unix(), 16)
	path := hash[:6] + SEP_S3_DIR + sep + SEP_S3_DIR + hash + SEP_SUFFIX + suffix
	pos := calcDistPosition(hash, dist)

	return path, pos
}

func calcDistPosition(hexhash string, dist int) int {
	l := len(hexhash)
	if l == 0 {
		return 0
	}

	c := hexhash[l-1 : l]
	h, ok := hexMap[c]
	if !ok {
		return 0
	}

	return h % dist
}

func resolvePosition(key string) (string, int, error) {
	idx := strings.LastIndex(key, SEP_POSITION)
	if idx == -1 {
		return "", -1, errors.New("key does not include position")
	}

	path := key[0:idx]
	posStr := key[idx+1 : len(key)]
	pos, err := strconv.Atoi(posStr)
	if err != nil {
		return "", -1, err
	}

	return path, pos, nil
}

func resolveS3Error(err error) error {
	// TODO smart handling aws-sdk-go error
	if idx := strings.Index(err.Error(), "status code: 404,"); idx != -1 {
		return ERR_FILE_NOT_FOUND
	}

	return err
}
