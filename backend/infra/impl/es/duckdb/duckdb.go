package duckdb

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	contract "github.com/coze-dev/coze-studio/backend/infra/contract/es"
)

// An in-memory minimal search client used in lite mode behind SEARCH_URI=duckdb://...
// It satisfies contract.Client without external dependencies. Text search is a simple substring match
// against a "text_content" field inside the document JSON.
type client struct {
	mu   sync.RWMutex
	path string
	// index -> id -> raw json
	docs map[string]map[string]json.RawMessage
}

func New(path string) contract.Client {
	c := &client{docs: make(map[string]map[string]json.RawMessage)}
	c.path = path
	if path != "" {
		// ensure parent dir
		if dir := filepath.Dir(path); dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}
		c.loadFromFile()
	}
	return c
}

func (c *client) loadFromFile() {
	if c.path == "" {
		return
	}
	f, err := os.Open(c.path)
	if err != nil {
		return
	}
	defer f.Close()
	dec := json.NewDecoder(f)
	var m map[string]map[string]json.RawMessage
	if err := dec.Decode(&m); err == nil && m != nil {
		c.docs = m
	}
}

func (c *client) saveToFileLocked() {
	if c.path == "" {
		return
	}
	tmp := c.path + ".tmp"
	if f, err := os.Create(tmp); err == nil {
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		_ = enc.Encode(c.docs)
		_ = f.Close()
		_ = os.Rename(tmp, c.path)
	}
}

func (c *client) Create(ctx context.Context, index, id string, document any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.docs[index]; !ok {
		c.docs[index] = map[string]json.RawMessage{}
	}
	b, _ := json.Marshal(document)
	c.docs[index][id] = b
	c.saveToFileLocked()
	return nil
}

func (c *client) Update(ctx context.Context, index, id string, document any) error {
	return c.Create(ctx, index, id, document)
}

func (c *client) Delete(ctx context.Context, index, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if m, ok := c.docs[index]; ok {
		delete(m, id)
	}
	c.saveToFileLocked()
	return nil
}

func (c *client) Search(ctx context.Context, index string, req *contract.Request) (*contract.Response, error) {
	_ = ctx
	c.mu.RLock()
	defer c.mu.RUnlock()

	m := c.docs[index]
	if m == nil {
		return &contract.Response{Hits: contract.HitsMetadata{Hits: nil}}, nil
	}

	// Collect matches
	type row struct {
		id    string
		raw   json.RawMessage
		obj   map[string]any
		score *float64
	}
	rows := make([]row, 0, len(m))

	// Legacy simple contains if explicit query is the KV-only with Value string and no key
	var legacyContains string
	if req != nil && req.Query != nil && req.Query.Type == "" && req.Query.Bool == nil && req.Query.KV.Value != nil && req.Query.KV.Key == "" {
		if s, ok := req.Query.KV.Value.(string); ok {
			legacyContains = strings.ToLower(s)
		}
	}

	for id, raw := range m {
		if legacyContains != "" {
			var obj map[string]any
			if err := json.Unmarshal(raw, &obj); err == nil {
				if v, ok := obj["text_content"].(string); ok && strings.Contains(strings.ToLower(v), legacyContains) {
					rows = append(rows, row{id: id, raw: raw, obj: obj})
				}
			}
			continue
		}

		// General query evaluation
		var obj map[string]any
		_ = json.Unmarshal(raw, &obj)
		var sc *float64
		matched := req == nil || req.Query == nil || matchQuery(obj, req.Query)
		// vector similarity: QueryTypeMatch with key 'embedding' → cosine similarity to query vector
		if req != nil && req.Query != nil && req.Query.Type == contract.QueryTypeMatch && req.Query.KV.Key == "embedding" {
			// Treat vector match as eligible by default; apply MinScore if provided
			matched = true
			if qvec, ok := toFloatSlice(req.Query.KV.Value); ok {
				if v, ok := obj["embedding"]; ok {
					if dvec, ok2 := toFloatSlice(v); ok2 {
						s := cosineSimilarity(qvec, dvec)
						sc = &s
						// apply MinScore filter if provided
						if req.MinScore != nil && *sc < *req.MinScore {
							matched = false
						}
					}
				}
			}
		}
		if matched {
			rows = append(rows, row{id: id, raw: raw, obj: obj, score: sc})
		}
	}

	// Sorting
	if req != nil && len(req.Sort) > 0 {
		sort.Slice(rows, func(i, j int) bool {
			// multi-field lexicographic sort
			for _, s := range req.Sort {
				if s.Field == "_score" {
					var si, sj float64
					if rows[i].score != nil {
						si = *rows[i].score
					}
					if rows[j].score != nil {
						sj = *rows[j].score
					}
					if si != sj {
						if s.Asc {
							return si < sj
						} else {
							return si > sj
						}
					}
					continue
				}
				vi, vj := rows[i].obj[s.Field], rows[j].obj[s.Field]
				cmp := compareAny(vi, vj)
				if cmp == 0 {
					continue
				}
				if s.Asc {
					return cmp < 0
				}
				return cmp > 0
			}
			return rows[i].id < rows[j].id
		})
	}

	// Pagination (size only)
	limit := len(rows)
	if req != nil && req.Size != nil && *req.Size >= 0 && *req.Size < limit {
		limit = *req.Size
	}

	hits := make([]contract.Hit, 0, limit)
	for k := 0; k < limit; k++ {
		idCopy := rows[k].id
		h := contract.Hit{Id_: &idCopy, Source_: rows[k].raw}
		if rows[k].score != nil {
			sc := *rows[k].score
			h.Score_ = &sc
		}
		hits = append(hits, h)
	}

	var maxScore *float64
	for _, r := range rows[:limit] {
		if r.score != nil {
			if maxScore == nil || *r.score > *maxScore {
				s := *r.score
				maxScore = &s
			}
		}
	}
	return &contract.Response{Hits: contract.HitsMetadata{Hits: hits, MaxScore: maxScore}}, nil
}

func getField(obj map[string]any, key string) (any, bool) {
	v, ok := obj[key]
	return v, ok
}

func toLowerString(v any) (string, bool) {
	if s, ok := v.(string); ok {
		return strings.ToLower(s), true
	}
	return "", false
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	case uint32:
		return float64(n), true
	default:
		return 0, false
	}
}

func toFloatSlice(v any) ([]float64, bool) {
	switch t := v.(type) {
	case []float64:
		return t, true
	case []any:
		out := make([]float64, 0, len(t))
		for _, it := range t {
			if f, ok := toFloat64(it); ok {
				out = append(out, f)
			} else {
				return nil, false
			}
		}
		return out, true
	default:
		return nil, false
	}
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	// naive cosine
	return dot / (sqrt(na) * sqrt(nb))
}

func sqrt(x float64) float64 {
	// simple Newton-Raphson for stability without importing math to keep build light
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 20; i++ {
		z = z - (z*z-x)/(2*z)
		if z <= 0 {
			z = 0.5
		}
	}
	return z
}

func compareAny(a, b any) int {
	// numeric compare if possible
	if fa, ok := toFloat64(a); ok {
		if fb, ok2 := toFloat64(b); ok2 {
			if fa < fb {
				return -1
			} else if fa > fb {
				return 1
			}
			return 0
		}
	}
	// string compare
	if sa, ok := a.(string); ok {
		if sb, ok2 := b.(string); ok2 {
			if sa < sb {
				return -1
			} else if sa > sb {
				return 1
			}
			return 0
		}
	}
	return 0
}

func matchQuery(obj map[string]any, q *contract.Query) bool {
	if q == nil {
		return true
	}
	// Bool
	if q.Bool != nil {
		// Filter (all must)
		for _, sub := range q.Bool.Filter {
			if !matchQuery(obj, &sub) {
				return false
			}
		}
		// Must (all)
		for _, sub := range q.Bool.Must {
			if !matchQuery(obj, &sub) {
				return false
			}
		}
		// MustNot (none)
		for _, sub := range q.Bool.MustNot {
			if matchQuery(obj, &sub) {
				return false
			}
		}
		// Should (at least MinimumShouldMatch or 1 if unspecified and there are shoulds)
		if len(q.Bool.Should) > 0 {
			required := 1
			if q.Bool.MinimumShouldMatch != nil {
				required = *q.Bool.MinimumShouldMatch
			}
			matched := 0
			for _, sub := range q.Bool.Should {
				if matchQuery(obj, &sub) {
					matched++
				}
			}
			if matched < required {
				return false
			}
		}
		return true
	}

	switch q.Type {
	case contract.QueryTypeEqual:
		val, ok := getField(obj, q.KV.Key)
		if !ok {
			return false
		}
		// numeric equality
		if fa, ok := toFloat64(val); ok {
			if fb, ok2 := toFloat64(q.KV.Value); ok2 {
				return fa == fb
			}
		}
		// string equality
		if sa, ok := val.(string); ok {
			if sb, ok2 := q.KV.Value.(string); ok2 {
				return sa == sb
			}
		}
		return false
	case contract.QueryTypeContains:
		val, ok := getField(obj, q.KV.Key)
		if !ok {
			return false
		}
		ls, ok1 := toLowerString(val)
		needle, ok2 := toLowerString(q.KV.Value)
		if !ok1 || !ok2 {
			return false
		}
		return strings.Contains(ls, needle)
	case contract.QueryTypeIn:
		val, ok := getField(obj, q.KV.Key)
		if !ok {
			return false
		}
		// numeric
		if fv, ok := toFloat64(val); ok {
			if arr, ok2 := q.KV.Value.([]any); ok2 {
				for _, it := range arr {
					if fi, ok3 := toFloat64(it); ok3 && fi == fv {
						return true
					}
				}
			}
			return false
		}
		// string
		if sv, ok := val.(string); ok {
			if arr, ok2 := q.KV.Value.([]any); ok2 {
				for _, it := range arr {
					if si, ok3 := it.(string); ok3 && si == sv {
						return true
					}
				}
			}
		}
		return false
	case contract.QueryTypeNotExists:
		_, ok := getField(obj, q.KV.Key)
		return !ok
	case contract.QueryTypeMatch, contract.QueryTypeMultiMatch:
		// Simplify to contains on the first field for match
		if q.KV.Key != "" {
			val, ok := getField(obj, q.KV.Key)
			if !ok {
				return false
			}
			ls, ok1 := toLowerString(val)
			needle, ok2 := toLowerString(q.KV.Value)
			if !ok1 || !ok2 {
				return false
			}
			return strings.Contains(ls, needle)
		}
		// MultiMatch: check any field in list contains query
		nq := strings.ToLower(q.MultiMatchQuery.Query)
		for _, f := range q.MultiMatchQuery.Fields {
			if v, ok := getField(obj, f); ok {
				if s, ok2 := v.(string); ok2 {
					if strings.Contains(strings.ToLower(s), nq) {
						return true
					}
				}
			}
		}
		return false
	default:
		// no query → match all
		return true
	}
}

func (c *client) Exists(ctx context.Context, index string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.docs[index]
	return ok, nil
}

func (c *client) CreateIndex(ctx context.Context, index string, _ map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.docs[index]; !ok {
		c.docs[index] = map[string]json.RawMessage{}
	}
	c.saveToFileLocked()
	return nil
}

func (c *client) DeleteIndex(ctx context.Context, index string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.docs, index)
	c.saveToFileLocked()
	return nil
}

func (c *client) Types() contract.Types { return &types{} }

type types struct{}

func (t *types) NewLongNumberProperty() any         { return struct{}{} }
func (t *types) NewTextProperty() any               { return struct{}{} }
func (t *types) NewUnsignedLongNumberProperty() any { return struct{}{} }

type bulk struct {
	c     *client
	index string
}

func (c *client) NewBulkIndexer(index string) (contract.BulkIndexer, error) {
	return &bulk{c: c, index: index}, nil
}
func (b *bulk) Add(ctx context.Context, item contract.BulkIndexerItem) error {
	var payload map[string]any
	if item.Body != nil {
		_ = json.NewDecoder(item.Body).Decode(&payload)
	}
	return b.c.Create(ctx, b.index, item.DocumentID, payload)
}
func (b *bulk) Close(ctx context.Context) error { return nil }
