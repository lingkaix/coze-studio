package redis

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBadger_BasicSetGetExpire(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("KV_URI", "badger://"+dir)
	t.Cleanup(func() { os.Unsetenv("KV_URI") })
	c := New()
	ctx := context.Background()
	require.NoError(t, c.Set(ctx, "k1", "v1", time.Second).Err())
	s, err := c.Get(ctx, "k1").Result()
	require.NoError(t, err)
	require.Equal(t, "v1", s)
}

func TestBadger_HashAndList(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("KV_URI", "badger://"+dir)
	t.Cleanup(func() { os.Unsetenv("KV_URI") })
	c := New()
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
	dir := t.TempDir()
	os.Setenv("KV_URI", "badger://"+dir)
	t.Cleanup(func() { os.Unsetenv("KV_URI") })
	c := New()
	ctx := context.Background()
	require.NoError(t, c.Incr(ctx, "cnt").Err())
	require.NoError(t, c.IncrBy(ctx, "cnt", 2).Err())
	v, err := c.Get(ctx, "cnt").Result()
	require.NoError(t, err)
	require.Equal(t, "3", v)
}
