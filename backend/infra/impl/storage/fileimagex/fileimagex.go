package fileimagex

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "net/url"
    "os"
    "strconv"
    "strings"
    "time"

    imgx "github.com/coze-dev/coze-studio/backend/infra/contract/imagex"
    "github.com/coze-dev/coze-studio/backend/pkg/ctxcache"
    "github.com/coze-dev/coze-studio/backend/types/consts"
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
    scheme, _ := ctxcache.Get[string](ctx, consts.RequestSchemeKeyInCtx)
    if scheme == "" {
        scheme = "http"
    }
    now := time.Now()
    return &imgx.SecurityToken{
        AccessKeyID:     "",
        SecretAccessKey: "",
        SessionToken:    "",
        ExpiredTime:     now.Add(time.Hour).Format("2006-01-02 15:04:05"),
        CurrentTime:     now.Format("2006-01-02 15:04:05"),
        HostScheme:      scheme,
    }, nil
}

func (c *client) GetUploadAuthWithExpire(ctx context.Context, expire time.Duration, opt ...imgx.UploadAuthOpt) (*imgx.SecurityToken, error) {
    t, _ := c.GetUploadAuth(ctx)
    now := time.Now()
    t.ExpiredTime = now.Add(expire).Format("2006-01-02 15:04:05")
    t.CurrentTime = now.Format("2006-01-02 15:04:05")
    return t, nil
}

func (c *client) GetResourceURL(ctx context.Context, uri string, opts ...imgx.GetResourceOpt) (*imgx.ResourceURL, error) {
    host, _ := ctxcache.Get[string](ctx, consts.HostKeyInCtx)
    scheme, _ := ctxcache.Get[string](ctx, consts.RequestSchemeKeyInCtx)
    if scheme == "" {
        scheme = "http"
    }

    trimmed := strings.TrimPrefix(uri, "/")
    expDur := time.Hour * 24
    if v := os.Getenv("FILE_SIGN_EXPIRE_SECONDS"); v != "" {
        if n, err := strconv.Atoi(v); err == nil && n > 0 {
            expDur = time.Duration(n) * time.Second
        }
    }
    exp := time.Now().Add(expDur).Unix()
    secret := os.Getenv("FILE_SIGN_SECRET")
    if secret == "" {
        secret = "dev-file-sign-secret"
    }
    sig := signHMAC(secret, trimmed, exp)

    u := url.URL{
        Scheme: scheme,
        Host:   host,
        Path:   "/api/common/blob/" + url.PathEscape(trimmed),
    }
    q := u.Query()
    q.Set("exp", strconv.FormatInt(exp, 10))
    q.Set("sig", sig)
    u.RawQuery = q.Encode()
    s := u.String()
    return &imgx.ResourceURL{URL: s, CompactURL: s}, nil
}

func (c *client) Upload(ctx context.Context, data []byte, opts ...imgx.UploadAuthOpt) (*imgx.UploadResult, error) {
    return &imgx.UploadResult{}, nil
}

func (c *client) GetServerID() string { return "local" }

func (c *client) GetUploadHost(ctx context.Context) string {
    host, _ := ctxcache.Get[string](ctx, consts.HostKeyInCtx)
    return host + consts.ApplyUploadActionURI
}

func signHMAC(secret, uri string, exp int64) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(uri))
    mac.Write([]byte("|"))
    mac.Write([]byte(strconv.FormatInt(exp, 10)))
    return hex.EncodeToString(mac.Sum(nil))
}


