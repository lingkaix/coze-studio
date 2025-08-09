package fileblob

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileBlob_PutGetDeleteAndURL(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()
	st, err := New(ctx, tmp)
	assert.NoError(t, err)

	key := "dir1/dir2/obj.txt"
	content := []byte("hello")

	err = st.PutObject(ctx, key, content)
	assert.NoError(t, err)

	data, err := st.GetObject(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, content, data)

	u, err := st.GetObjectUrl(ctx, key)
	assert.NoError(t, err)
	assert.Contains(t, u, "file:")

	// file must physically exist
	abs := filepath.Join(tmp, filepath.FromSlash(key))
	_, statErr := os.Stat(abs)
	assert.NoError(t, statErr)

	// delete
	err = st.DeleteObject(ctx, key)
	assert.NoError(t, err)

	_, statErr = os.Stat(abs)
	assert.Error(t, statErr)
}
