package badger

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/coze-dev/coze-studio/backend/infra/contract/cache"
	badger "github.com/dgraph-io/badger/v4"
)

type badgerImpl struct {
	db *badger.DB
}

// New creates a Badger-backed cache.Cmdable at the provided directory path.
// Callers are responsible for passing a per-test temp dir in tests.
func New(path string) (cache.Cmdable, error) {
	if path == "" {
		path = "./var/data/badger"
	}
	opts := badger.DefaultOptions(filepath.Clean(path))
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &badgerImpl{db: db}, nil
}

// Helpers
func keyFor(k string) []byte     { return []byte("k:" + k) }
func hashKeyFor(k string) []byte { return []byte("h:" + k + ":") }
func listKeyFor(k string) []byte { return []byte("l:" + k + ":") }

func (b *badgerImpl) withUpdate(fn func(*badger.Txn) error) error {
	return b.db.Update(fn)
}
func (b *badgerImpl) withView(fn func(*badger.Txn) error) error { return b.db.View(fn) }

// base cmd wrappers
type errOnly struct{ err error }

func (e errOnly) Err() error { return e.err }

type intRes struct {
	err error
	v   int64
}

func (r intRes) Err() error             { return r.err }
func (r intRes) Result() (int64, error) { return r.v, r.err }

type statusRes struct {
	err error
	s   string
}

func (r statusRes) Err() error              { return r.err }
func (r statusRes) Result() (string, error) { return r.s, r.err }

type boolRes struct {
	err error
	v   bool
}

func (r boolRes) Err() error            { return r.err }
func (r boolRes) Result() (bool, error) { return r.v, r.err }

type strRes struct {
	err error
	v   []byte
}

func (r strRes) Err() error              { return r.err }
func (r strRes) Result() (string, error) { return string(r.v), r.err }
func (r strRes) Val() string             { s, _ := r.Result(); return s }
func (r strRes) Int64() (int64, error)   { return 0, errors.New("not supported") }
func (r strRes) Bytes() ([]byte, error)  { return r.v, r.err }

type mapStrStrRes struct {
	err error
	m   map[string]string
}

func (r mapStrStrRes) Err() error                         { return r.err }
func (r mapStrStrRes) Result() (map[string]string, error) { return r.m, r.err }

type strSliceRes struct {
	err error
	v   []string
}

func (r strSliceRes) Err() error                { return r.err }
func (r strSliceRes) Result() ([]string, error) { return r.v, r.err }

// StringCmdable
func (b *badgerImpl) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) cache.StatusCmd {
	_ = ctx
	bs := []byte(fmt.Sprint(value))
	err := b.withUpdate(func(txn *badger.Txn) error {
		e := badger.NewEntry(keyFor(key), bs)
		if expiration > 0 {
			e = e.WithTTL(expiration)
		}
		return txn.SetEntry(e)
	})
	return statusRes{err: err, s: "OK"}
}

func (b *badgerImpl) Get(ctx context.Context, key string) cache.StringCmd {
	_ = ctx
	var out []byte
	err := b.withView(func(txn *badger.Txn) error {
		it, err := txn.Get(keyFor(key))
		if err != nil {
			return err
		}
		return it.Value(func(val []byte) error { out = append([]byte{}, val...); return nil })
	})
	if err == badger.ErrKeyNotFound {
		return strRes{err: cache.Nil}
	}
	return strRes{err: err, v: out}
}

func (b *badgerImpl) IncrBy(ctx context.Context, key string, value int64) cache.IntCmd {
	_ = ctx
	var res int64
	err := b.withUpdate(func(txn *badger.Txn) error {
		var cur int64
		if item, err := txn.Get(keyFor(key)); err == nil {
			_ = item.Value(func(val []byte) error { _, _ = fmt.Sscan(string(val), &cur); return nil })
		} else if err != badger.ErrKeyNotFound {
			return err
		}
		res = cur + value
		return txn.Set(keyFor(key), []byte(fmt.Sprint(res)))
	})
	return intRes{err: err, v: res}
}

func (b *badgerImpl) Incr(ctx context.Context, key string) cache.IntCmd { return b.IncrBy(ctx, key, 1) }

// HashCmdable
func (b *badgerImpl) HSet(ctx context.Context, key string, values ...interface{}) cache.IntCmd {
	_ = ctx
	if len(values)%2 != 0 {
		return intRes{err: errors.New("odd arguments")}
	}
	var count int64
	err := b.withUpdate(func(txn *badger.Txn) error {
		prefix := hashKeyFor(key)
		for i := 0; i < len(values); i += 2 {
			field := fmt.Sprint(values[i])
			val := []byte(fmt.Sprint(values[i+1]))
			// allocate new key buffer per field to avoid mutating prefix
			k := make([]byte, len(prefix))
			copy(k, prefix)
			k = append(k, []byte(field)...)
			if err := txn.Set(k, val); err != nil {
				return err
			}
			count++
		}
		return nil
	})
	return intRes{err: err, v: count}
}

func (b *badgerImpl) HGetAll(ctx context.Context, key string) cache.MapStringStringCmd {
	_ = ctx
	result := map[string]string{}
	err := b.withView(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := hashKeyFor(key)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			k := item.Key()
			field := strings.TrimPrefix(string(k), string(prefix))
			_ = item.Value(func(v []byte) error { result[field] = string(v); return nil })
		}
		return nil
	})
	return mapStrStrRes{err: err, m: result}
}

// GenericCmdable
func (b *badgerImpl) Del(ctx context.Context, keys ...string) cache.IntCmd {
	_ = ctx
	var n int64
	err := b.withUpdate(func(txn *badger.Txn) error {
		for _, k := range keys {
			if err := txn.Delete(keyFor(k)); err == nil {
				n++
			}
		}
		return nil
	})
	return intRes{err: err, v: n}
}

func (b *badgerImpl) Exists(ctx context.Context, keys ...string) cache.IntCmd {
	_ = ctx
	var n int64
	_ = b.withView(func(txn *badger.Txn) error {
		for _, k := range keys {
			if _, err := txn.Get(keyFor(k)); err == nil {
				n++
			}
		}
		return nil
	})
	return intRes{v: n}
}

func (b *badgerImpl) Expire(ctx context.Context, key string, expiration time.Duration) cache.BoolCmd {
	_ = ctx
	var ok bool
	err := b.withUpdate(func(txn *badger.Txn) error {
		it, err := txn.Get(keyFor(key))
		if err != nil {
			return err
		}
		var val []byte
		_ = it.Value(func(v []byte) error { val = append([]byte{}, v...); return nil })
		e := badger.NewEntry(keyFor(key), val).WithTTL(expiration)
		if err := txn.SetEntry(e); err != nil {
			return err
		}
		ok = true
		return nil
	})
	if err == badger.ErrKeyNotFound {
		return boolRes{v: false}
	}
	return boolRes{err: err, v: ok}
}

// ListCmdable (simplified; stores sequential indexes)
func (b *badgerImpl) LPush(ctx context.Context, key string, values ...interface{}) cache.IntCmd {
	_ = ctx
	var length int64
	err := b.withUpdate(func(txn *badger.Txn) error {
		// read all existing
		cur, _ := b.listReadAll(txn, key)
		// prepend
		for i := len(values) - 1; i >= 0; i-- {
			cur = append([]string{fmt.Sprint(values[i])}, cur...)
		}
		length = int64(len(cur))
		return b.listWriteAll(txn, key, cur)
	})
	return intRes{err: err, v: length}
}

func (b *badgerImpl) RPush(ctx context.Context, key string, values ...interface{}) cache.IntCmd {
	_ = ctx
	var length int64
	err := b.withUpdate(func(txn *badger.Txn) error {
		cur, _ := b.listReadAll(txn, key)
		for _, v := range values {
			cur = append(cur, fmt.Sprint(v))
		}
		length = int64(len(cur))
		return b.listWriteAll(txn, key, cur)
	})
	return intRes{err: err, v: length}
}

func (b *badgerImpl) LPop(ctx context.Context, key string) cache.StringCmd {
	_ = ctx
	var out string
	err := b.withUpdate(func(txn *badger.Txn) error {
		cur, _ := b.listReadAll(txn, key)
		if len(cur) == 0 {
			return cache.Nil
		}
		out = cur[0]
		cur = cur[1:]
		return b.listWriteAll(txn, key, cur)
	})
	if err == cache.Nil {
		return strRes{err: cache.Nil}
	}
	return strRes{err: err, v: []byte(out)}
}

func (b *badgerImpl) LIndex(ctx context.Context, key string, index int64) cache.StringCmd {
	_ = ctx
	var out string
	err := b.withView(func(txn *badger.Txn) error {
		cur, _ := b.listReadAll(txn, key)
		if index < 0 || index >= int64(len(cur)) {
			return cache.Nil
		}
		out = cur[index]
		return nil
	})
	if err == cache.Nil {
		return strRes{err: cache.Nil}
	}
	return strRes{err: err, v: []byte(out)}
}

func (b *badgerImpl) LRange(ctx context.Context, key string, start, stop int64) cache.StringSliceCmd {
	_ = ctx
	var out []string
	err := b.withView(func(txn *badger.Txn) error {
		cur, _ := b.listReadAll(txn, key)
		if stop < 0 || stop >= int64(len(cur)) {
			stop = int64(len(cur)) - 1
		}
		if start < 0 {
			start = 0
		}
		if start > stop || len(cur) == 0 {
			out = []string{}
			return nil
		}
		out = append([]string{}, cur[start:stop+1]...)
		return nil
	})
	return strSliceRes{err: err, v: out}
}

func (b *badgerImpl) LSet(ctx context.Context, key string, index int64, value interface{}) cache.StatusCmd {
	_ = ctx
	err := b.withUpdate(func(txn *badger.Txn) error {
		cur, _ := b.listReadAll(txn, key)
		if index < 0 || index >= int64(len(cur)) {
			return cache.Nil
		}
		cur[index] = fmt.Sprint(value)
		return b.listWriteAll(txn, key, cur)
	})
	if err == cache.Nil {
		return statusRes{err: cache.Nil}
	}
	return statusRes{err: err, s: "OK"}
}

func (b *badgerImpl) listReadAll(txn *badger.Txn, key string) ([]string, error) {
	prefix := listKeyFor(key)
	opts := badger.DefaultIteratorOptions
	it := txn.NewIterator(opts)
	defer it.Close()
	out := []string{}
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		_ = item.Value(func(v []byte) error { out = append(out, string(v)); return nil })
	}
	return out, nil
}

func (b *badgerImpl) listWriteAll(txn *badger.Txn, key string, arr []string) error {
	// delete old
	prefix := listKeyFor(key)
	opts := badger.DefaultIteratorOptions
	it := txn.NewIterator(opts)
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		_ = txn.Delete(it.Item().Key())
	}
	it.Close()
	// write sequential
	for idx, val := range arr {
		k := make([]byte, len(prefix))
		copy(k, prefix)
		k = append(k, []byte(fmt.Sprintf("%08d", idx))...)
		if err := txn.Set(k, []byte(val)); err != nil {
			return err
		}
	}
	return nil
}

// Pipeline: buffer ops and apply in one Update
type badgerPipe struct {
	b   *badgerImpl
	ops []func(*badger.Txn) error
}

func (p *badgerPipe) Pipeline() cache.Pipeliner { return p }
func (p *badgerPipe) Exec(ctx context.Context) ([]cache.Cmder, error) {
	_ = ctx
	if len(p.ops) == 0 {
		return nil, nil
	}
	var firstErr error
	err := p.b.withUpdate(func(txn *badger.Txn) error {
		for _, op := range p.ops {
			if e := op(txn); e != nil && firstErr == nil {
				firstErr = e
			}
		}
		return nil
	})
	if err != nil && firstErr == nil {
		firstErr = err
	}
	// For simplicity return empty cmder list; callers check errors via Exec
	return nil, firstErr
}

func (b *badgerImpl) Pipeline() cache.Pipeliner { return &badgerPipe{b: b} }

// Implement Cmdable methods on pipeline by forwarding to underlying immediately
func (p *badgerPipe) Del(ctx context.Context, keys ...string) cache.IntCmd {
	return p.b.Del(ctx, keys...)
}
func (p *badgerPipe) Exists(ctx context.Context, keys ...string) cache.IntCmd {
	return p.b.Exists(ctx, keys...)
}
func (p *badgerPipe) Expire(ctx context.Context, key string, expiration time.Duration) cache.BoolCmd {
	return p.b.Expire(ctx, key, expiration)
}
func (p *badgerPipe) Get(ctx context.Context, key string) cache.StringCmd { return p.b.Get(ctx, key) }
func (p *badgerPipe) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) cache.StatusCmd {
	return p.b.Set(ctx, key, value, expiration)
}
func (p *badgerPipe) Incr(ctx context.Context, key string) cache.IntCmd { return p.b.Incr(ctx, key) }
func (p *badgerPipe) IncrBy(ctx context.Context, key string, value int64) cache.IntCmd {
	return p.b.IncrBy(ctx, key, value)
}
func (p *badgerPipe) HSet(ctx context.Context, key string, values ...interface{}) cache.IntCmd {
	return p.b.HSet(ctx, key, values...)
}
func (p *badgerPipe) HGetAll(ctx context.Context, key string) cache.MapStringStringCmd {
	return p.b.HGetAll(ctx, key)
}
func (p *badgerPipe) LIndex(ctx context.Context, key string, index int64) cache.StringCmd {
	return p.b.LIndex(ctx, key, index)
}
func (p *badgerPipe) LPop(ctx context.Context, key string) cache.StringCmd { return p.b.LPop(ctx, key) }
func (p *badgerPipe) LPush(ctx context.Context, key string, values ...interface{}) cache.IntCmd {
	return p.b.LPush(ctx, key, values...)
}
func (p *badgerPipe) LRange(ctx context.Context, key string, start, stop int64) cache.StringSliceCmd {
	return p.b.LRange(ctx, key, start, stop)
}
func (p *badgerPipe) LSet(ctx context.Context, key string, index int64, value interface{}) cache.StatusCmd {
	return p.b.LSet(ctx, key, index, value)
}
func (p *badgerPipe) RPush(ctx context.Context, key string, values ...interface{}) cache.IntCmd {
	return p.b.RPush(ctx, key, values...)
}
