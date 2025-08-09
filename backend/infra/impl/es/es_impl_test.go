package es

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFactory_SelectsDuckDBInLiteWithSearchURI(t *testing.T) {
	// Ensure clean env and enable lite + duckdb
	os.Setenv("LITE_MODE", "1")
	dir := t.TempDir()
	os.Setenv("SEARCH_URI", "duckdb://"+dir+"/test.duckdb")
	t.Cleanup(func() {
		os.Unsetenv("LITE_MODE")
		os.Unsetenv("SEARCH_URI")
	})

	cli, err := New()
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, cli.CreateIndex(ctx, "t1", nil))
	require.NoError(t, cli.Create(ctx, "t1", "id1", map[string]any{"text_content": "hello lite duck"}))

	resp, err := cli.Search(ctx, "t1", &Request{Query: &Query{KV: struct {
		Key   string
		Value any
	}{Value: "duck"}}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(resp.Hits.Hits), 1)
}
