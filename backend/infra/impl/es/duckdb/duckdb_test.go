package duckdb

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	contract "github.com/coze-dev/coze-studio/backend/infra/contract/es"
	"github.com/stretchr/testify/require"
)

func TestDuckDBClient_BasicCRUDAndSearch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	c := New(filepath.Join(dir, "test.duckdb.json"))
	ctx := context.Background()

	// Create index
	require.NoError(t, c.CreateIndex(ctx, "docs", nil))
	exist, err := c.Exists(ctx, "docs")
	require.NoError(t, err)
	require.True(t, exist)

	// Index two docs
	require.NoError(t, c.Create(ctx, "docs", "1", map[string]any{"text_content": "Hello World"}))
	require.NoError(t, c.Create(ctx, "docs", "2", map[string]any{"text_content": "another line"}))

	// Bulk index a third
	bi, err := c.NewBulkIndexer("docs")
	require.NoError(t, err)
	body, _ := json.Marshal(map[string]any{"text_content": "HELLO from bulk"})
	require.NoError(t, bi.Add(ctx, contract.BulkIndexerItem{DocumentID: "3", Body: bytes.NewReader(body)}))
	require.NoError(t, bi.Close(ctx))

	// Search for hello (case-insensitive contains)
	resp, err := c.Search(ctx, "docs", &contract.Request{Query: &contract.Query{KV: contract.KV{Value: "hello"}}})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(resp.Hits.Hits), 2)

	// Delete and confirm
	require.NoError(t, c.Delete(ctx, "docs", "2"))
	resp2, err := c.Search(ctx, "docs", &contract.Request{Query: &contract.Query{KV: contract.KV{Value: "another"}}})
	require.NoError(t, err)
	require.Equal(t, 0, len(resp2.Hits.Hits))

	// Delete index
	require.NoError(t, c.DeleteIndex(ctx, "docs"))
	exist, err = c.Exists(ctx, "docs")
	require.NoError(t, err)
	require.False(t, exist)
}

func TestDuckDBClient_Persistence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "persist.duckdb.json")
	// First instance writes data
	c1 := New(path)
	ctx := context.Background()
	require.NoError(t, c1.CreateIndex(ctx, "docs", nil))
	require.NoError(t, c1.Create(ctx, "docs", "1", map[string]any{"text_content": "persist me"}))

	// New instance loads and can search
	c2 := New(path)
	resp, err := c2.Search(ctx, "docs", &contract.Request{Query: &contract.Query{KV: contract.KV{Value: "persist"}}})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Hits.Hits))
}

func TestDuckDBClient_QueriesAndSorting(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	c := New(filepath.Join(dir, "query.duckdb.json"))
	ctx := context.Background()
	require.NoError(t, c.CreateIndex(ctx, "docs", nil))

	// seed docs
	require.NoError(t, c.Create(ctx, "docs", "a", map[string]any{"text_content": "hello world", "category": "news", "views": 10}))
	require.NoError(t, c.Create(ctx, "docs", "b", map[string]any{"text_content": "HELLO golang", "category": "blog", "views": 50}))
	require.NoError(t, c.Create(ctx, "docs", "c", map[string]any{"text_content": "random", "category": "news", "views": 30}))

	// equal on category
	qEqual := contract.NewEqualQuery("category", "news")
	resp, err := c.Search(ctx, "docs", &contract.Request{Query: &qEqual})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.Hits.Hits))

	// contains on text_content
	qContains := contract.NewContainsQuery("text_content", "hello")
	resp, err = c.Search(ctx, "docs", &contract.Request{Query: &qContains})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.Hits.Hits))

	// bool must: category=news and contains random
	must := []contract.Query{contract.NewEqualQuery("category", "news"), contract.NewContainsQuery("text_content", "random")}
	qBool := contract.Query{Bool: &contract.BoolQuery{Must: must}}
	resp, err = c.Search(ctx, "docs", &contract.Request{Query: &qBool})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Hits.Hits))

	// sort by views desc, take top1
	size := 1
	resp, err = c.Search(ctx, "docs", &contract.Request{Query: &contract.Query{}, Size: &size, Sort: []contract.SortFiled{{Field: "views", Asc: false}}})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Hits.Hits))
}

func TestDuckDBClient_VectorSimilarity(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	c := New(filepath.Join(dir, "vector.duckdb.json"))
	ctx := context.Background()
	require.NoError(t, c.CreateIndex(ctx, "docs", nil))

	// two docs with embeddings
	require.NoError(t, c.Create(ctx, "docs", "v1", map[string]any{"embedding": []float64{1, 0, 0}, "label": "x"}))
	require.NoError(t, c.Create(ctx, "docs", "v2", map[string]any{"embedding": []float64{0.8, 0.2, 0}, "label": "y"}))
	// query vector close to v2
	qvec := []float64{0.8, 0.2, 0}
	size := 1
	minScore := 0.5
	resp, err := c.Search(ctx, "docs", &contract.Request{
		Query:    &contract.Query{KV: contract.KV{Key: "embedding", Value: qvec}, Type: contract.QueryTypeMatch},
		MinScore: &minScore,
		Size:     &size,
		Sort:     []contract.SortFiled{{Field: "_score", Asc: false}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Hits.Hits))
	require.NotNil(t, resp.Hits.MaxScore)
}
