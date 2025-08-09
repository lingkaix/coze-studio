package fileblob

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	storagecontract "github.com/coze-dev/coze-studio/backend/infra/contract/storage"
	"gocloud.dev/blob"
	gocdkfileblob "gocloud.dev/blob/fileblob"
)

type client struct {
	bucket  *blob.Bucket
	baseDir string
}

// New creates a file-backed storage client rooted at baseDir.
func New(ctx context.Context, baseDir string) (storagecontract.Storage, error) {
	if baseDir == "" {
		baseDir = "./var/data/blob"
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("abs base dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}
	b, err := gocdkfileblob.OpenBucket(abs, &gocdkfileblob.Options{CreateDir: true})
	if err != nil {
		return nil, fmt.Errorf("open fileblob bucket: %w", err)
	}
	return &client{bucket: b, baseDir: abs}, nil
}

func (c *client) PutObject(ctx context.Context, objectKey string, content []byte, _ ...storagecontract.PutOptFn) error {
	w, err := c.bucket.NewWriter(ctx, objectKey, nil)
	if err != nil {
		return err
	}
	if _, err := w.Write(content); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

func (c *client) GetObject(ctx context.Context, objectKey string) ([]byte, error) {
	r, err := c.bucket.NewReader(ctx, objectKey, nil)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (c *client) DeleteObject(ctx context.Context, objectKey string) error {
	return c.bucket.Delete(ctx, objectKey)
}

func (c *client) GetObjectUrl(_ context.Context, objectKey string, _ ...storagecontract.GetOptFn) (string, error) {
	// Return a file:// URL pointing to the absolute path
	p := filepath.Join(c.baseDir, filepath.FromSlash(objectKey))
	u := url.URL{Scheme: "file", Path: p}
	return u.String(), nil
}

// Optional helpers for options compatibility
type putOptions struct{}

type getURLOptions struct{ Expire time.Duration }
