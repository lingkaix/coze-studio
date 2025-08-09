package badger

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBadger_BasicSetGetExpire(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k1", "v1", time.Second).Err())
	s, err := c.Get(ctx, "k1").Result()
	require.NoError(t, err)
	require.Equal(t, "v1", s)

	// also exercise Expire
	require.NoError(t, c.Expire(ctx, "k1", time.Second).Err())
}

func TestBadger_HashAndList(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, c.HSet(ctx, "h1", "f1", "a", "f2", "b").Err())
	m, err := c.HGetAll(ctx, "h1").Result()
	require.NoError(t, err)
	require.Equal(t, "a", m["f1"])

	require.NoError(t, c.RPush(ctx, "l1", "x", "y").Err())
	arr, err := c.LRange(ctx, "l1", 0, -1).Result()
	require.NoError(t, err)
	require.Equal(t, []string{"x", "y"}, arr)
}

func TestBadger_Incr(t *testing.T) {
	_ = filepath.Base("") // keep filepath imported even if not used when editing
	dir := t.TempDir()
	c, err := New(dir)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, c.Incr(ctx, "cnt").Err())
	require.NoError(t, c.IncrBy(ctx, "cnt", 2).Err())
	v, err := c.Get(ctx, "cnt").Result()
	require.NoError(t, err)
	require.Equal(t, "3", v)
}
