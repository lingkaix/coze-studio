package coze

import (
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "time"

    "github.com/cloudwego/hertz/pkg/app"
)

// BlobDownload serves HMAC-signed local files in file mode.
// Route: GET /api/common/blob/*uri?exp=...&sig=...
func BlobDownload(ctx context.Context, c *app.RequestContext) {
    rawURI := c.Param("uri")
    if len(rawURI) > 0 && rawURI[0] == '/' {
        rawURI = rawURI[1:]
    }

    expStr := string(c.Query("exp"))
    sig := string(c.Query("sig"))

    if expStr == "" || sig == "" || rawURI == "" {
        c.AbortWithStatus(http.StatusForbidden)
        return
    }

    exp, err := strconv.ParseInt(expStr, 10, 64)
    if err != nil || time.Now().Unix() > exp {
        c.AbortWithStatus(http.StatusForbidden)
        return
    }

    secret := os.Getenv("FILE_SIGN_SECRET")
    if secret == "" {
        secret = "dev-file-sign-secret"
    }

    if sig != hmacSign(secret, rawURI, exp) {
        c.AbortWithStatus(http.StatusForbidden)
        return
    }

    baseDir := os.Getenv("BLOB_DIR")
    if baseDir == "" {
        baseDir = "./var/data/blob"
    }
    fp := filepath.Join(baseDir, filepath.FromSlash(rawURI))

    f, err := os.Open(fp)
    if err != nil {
        c.AbortWithStatus(http.StatusNotFound)
        return
    }
    defer f.Close()

    // naive content-type detection
    header := make([]byte, 512)
    n, _ := f.Read(header)
    contentType := http.DetectContentType(header[:n])
    _, _ = f.Seek(0, 0)

    c.Response.Header.Set("Content-Type", contentType)
    c.SetStatusCode(http.StatusOK)
    _, _ = io.Copy(c, f)
}

func hmacSign(secret string, uri string, exp int64) string {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(uri))
    mac.Write([]byte("|"))
    mac.Write([]byte(strconv.FormatInt(exp, 10)))
    return hex.EncodeToString(mac.Sum(nil))
}


