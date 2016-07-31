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

	RNBIN_META_ORIGIN_NAME = "Rnbin-Origin-Name"
	RNBIN_META_CREATED_BY  = "Rnbin-Created-By"
	RNBIN_META_SEP         = "Rnbin-Sep"
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
	OriginName  string
	ContentType string
	Data        []byte
	Sep         string
	CreatedBy   string
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
	return s3.StoreWithReader(pos, data.Sep, path, data.OriginName, data.ContentType, data.CreatedBy, r)
}

func (s3 *S3Backend) StoreWithReader(pos int, sep, path, name, contentType, createdBy string, reader io.Reader) (string, error) {
	// UploadInput
	// https://github.com/aws/aws-sdk-go/blob/master/service/s3/s3manager/upload.go#L99

	bucket, err := s3.GetBucketName(pos)
	if err != nil {
		return "", err
	}

	meta := make(map[string]*string, 3)

	meta[RNBIN_META_ORIGIN_NAME] = &name
	meta[RNBIN_META_CREATED_BY] = &createdBy
	meta[RNBIN_META_SEP] = &sep
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

func (s3m *S3Backend) GetMeta(key string) (*Meta, error) {
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

	return convertMeta(resp), nil
}

type Meta struct {
	ContentType   string
	ContentLength int64
	LastModified  time.Time

	OriginName string
	Sep        string
	CreatedBy  string
}

func convertMeta(resp *s3.HeadObjectOutput) *Meta {
	respMeta := Meta{}
	ct := resp.ContentType
	if ct != nil {
		respMeta.ContentType = *ct
	}

	cl := resp.ContentLength
	if cl != nil {
		respMeta.ContentLength = *cl
	}

	lm := resp.LastModified
	if lm != nil {
		respMeta.LastModified = *lm
	}

	s3meta := resp.Metadata
	name := s3meta[RNBIN_META_ORIGIN_NAME]
	if name != nil && *name != "" {
		respMeta.OriginName = *name
	}

	cb := s3meta[RNBIN_META_CREATED_BY]
	if cb != nil && *cb != "" {
		respMeta.CreatedBy = *cb
	}

	sep := s3meta[RNBIN_META_SEP]
	if sep != nil && *sep != "" {
		respMeta.Sep = *sep
	}

	return &respMeta
}

func convertMetaObject(resp *s3.GetObjectOutput) *Meta {
	respMeta := Meta{}
	ct := resp.ContentType
	if ct != nil {
		respMeta.ContentType = *ct
	}

	cl := resp.ContentLength
	if cl != nil {
		respMeta.ContentLength = *cl
	}

	lm := resp.LastModified
	if lm != nil {
		respMeta.LastModified = *lm
	}

	s3meta := resp.Metadata
	sep := s3meta["Rnbin-Sep"]
	if sep != nil && *sep != "" {
		respMeta.Sep = *sep
	}

	cb := s3meta["Rnbin-Createdby"]
	if cb != nil && *cb != "" {
		respMeta.CreatedBy = *cb
	}

	return &respMeta
}

func (s3m *S3Backend) GetObject(key string) ([]byte, *Meta, error) {
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

func (s3m *S3Backend) GetObjectWithReadCloser(pos int, path string) (io.ReadCloser, *Meta, error) {
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

	return resp.Body, convertMetaObject(resp), nil
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
