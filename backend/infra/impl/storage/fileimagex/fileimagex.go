package fileimagex

import (
    "context"
    "net/url"
    "path/filepath"
    "strings"
    "time"

    imgx "github.com/coze-dev/coze-studio/backend/infra/contract/imagex"
)

type client struct {
    baseDir string
}

func New(baseDir string) (imgx.ImageX, error) {
    if baseDir == "" {
        baseDir = "./var/data/blob"
    }
    return &client{baseDir: baseDir}, nil
}

func (c *client) GetUploadAuth(ctx context.Context, opt ...imgx.UploadAuthOpt) (*imgx.SecurityToken, error) {
    return &imgx.SecurityToken{HostScheme: "file"}, nil
}

func (c *client) GetUploadAuthWithExpire(ctx context.Context, expire time.Duration, opt ...imgx.UploadAuthOpt) (*imgx.SecurityToken, error) {
    return &imgx.SecurityToken{HostScheme: "file"}, nil
}

func (c *client) GetResourceURL(ctx context.Context, uri string, opts ...imgx.GetResourceOpt) (*imgx.ResourceURL, error) {
    // Construct file:// URL relative to baseDir
    p := filepath.Join(c.baseDir, filepath.FromSlash(strings.TrimPrefix(uri, "/")))
    u := url.URL{Scheme: "file", Path: p}
    s := u.String()
    return &imgx.ResourceURL{URL: s, CompactURL: s}, nil
}

func (c *client) Upload(ctx context.Context, data []byte, opts ...imgx.UploadAuthOpt) (*imgx.UploadResult, error) {
    // Not supported for local file imagex stub; callers in lite mode should use Storage directly
    return &imgx.UploadResult{}, nil
}

func (c *client) GetServerID() string { return "local" }
func (c *client) GetUploadHost(ctx context.Context) string { return "file" }


