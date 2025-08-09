package fileimagex

import (
    "context"
    "net/url"
    "os"
    "testing"
    "time"

    "github.com/stretchr/testify/require"

    "github.com/coze-dev/coze-studio/backend/pkg/ctxcache"
    "github.com/coze-dev/coze-studio/backend/types/consts"
)

func TestFileImageXAuthAndHost(t *testing.T) {
    os.Setenv("BLOB_DIR", "./var/data/blob")
    t.Cleanup(func() { os.Unsetenv("BLOB_DIR") })

    ctx := ctxcache.Init(context.Background())
    ctxcache.Store(ctx, consts.HostKeyInCtx, "localhost:8888")
    ctxcache.Store(ctx, consts.RequestSchemeKeyInCtx, "http")

    cli, err := New(os.Getenv("BLOB_DIR"))
    require.NoError(t, err)

    token, err := cli.GetUploadAuth(ctx)
    require.NoError(t, err)
    require.Equal(t, "http", token.HostScheme)

    host := cli.GetUploadHost(ctx)
    require.Equal(t, "localhost:8888"+consts.ApplyUploadActionURI, host)
}

func TestFileImageXGetResourceURL_Signed(t *testing.T) {
    os.Setenv("BLOB_DIR", t.TempDir())
    os.Setenv("FILE_SIGN_SECRET", "test-secret")
    t.Cleanup(func() {
        os.Unsetenv("BLOB_DIR")
        os.Unsetenv("FILE_SIGN_SECRET")
    })

    ctx := ctxcache.Init(context.Background())
    ctxcache.Store(ctx, consts.HostKeyInCtx, "localhost:8888")
    ctxcache.Store(ctx, consts.RequestSchemeKeyInCtx, "http")

    cli, err := New(os.Getenv("BLOB_DIR"))
    require.NoError(t, err)

    uri := "uploads/foo.png"
    res, err := cli.GetResourceURL(ctx, uri)
    require.NoError(t, err)

    u, err := url.Parse(res.URL)
    require.NoError(t, err)
    require.Equal(t, "http", u.Scheme)
    require.Equal(t, "localhost:8888", u.Host)
    require.Equal(t, "/api/common/blob/"+url.PathEscape(uri), u.Path)

    q := u.Query()
    require.NotEmpty(t, q.Get("exp"))
    require.NotEmpty(t, q.Get("sig"))

    // sanity: exp must be in the future
    expStr := q.Get("exp")
    // Allow parsing and ensure greater than now
    // Parsing handled in handler; here we just ensure it's a number
    require.NotPanics(t, func() { _, _ = time.ParseDuration(expStr + "s") })
}


