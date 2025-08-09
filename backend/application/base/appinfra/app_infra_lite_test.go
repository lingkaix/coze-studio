package appinfra

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyLitePreset_SetsDefaults(t *testing.T) {
	// Avoid t.Parallel due to process-wide env mutations
	// clear related envs
	os.Unsetenv("LITE_MODE")
	os.Unsetenv("COZE_LITE")
	os.Unsetenv("DB_URI")
	os.Unsetenv("COZE_MQ_TYPE")
	os.Unsetenv("STORAGE_TYPE")
	os.Unsetenv("BLOB_DIR")
	os.Unsetenv("CACHE_URI")
	os.Unsetenv("SEARCH_URI")

	// enable lite
	os.Setenv("LITE_MODE", "1")
	t.Cleanup(func() {
		os.Unsetenv("LITE_MODE")
		os.Unsetenv("COZE_LITE")
		os.Unsetenv("DB_URI")
		os.Unsetenv("COZE_MQ_TYPE")
		os.Unsetenv("STORAGE_TYPE")
		os.Unsetenv("BLOB_DIR")
		os.Unsetenv("CACHE_URI")
		os.Unsetenv("SEARCH_URI")
	})

	applyLitePreset()

	assert.Equal(t, "mem", os.Getenv("COZE_MQ_TYPE"))
	assert.NotEmpty(t, os.Getenv("DB_URI"))
	assert.Equal(t, "file", os.Getenv("STORAGE_TYPE"))
	assert.NotEmpty(t, os.Getenv("BLOB_DIR"))
	// KV preset present (either KV_URI or CACHE_URI)
	if os.Getenv("KV_URI") == "" {
		assert.NotEmpty(t, os.Getenv("CACHE_URI"))
	}
	assert.NotEmpty(t, os.Getenv("SEARCH_URI"))
}

func TestApplyLitePreset_DoesNotOverrideExisting(t *testing.T) {
	// Avoid t.Parallel due to process-wide env mutations
	os.Setenv("LITE_MODE", "1")
	os.Setenv("DB_URI", "sqlite:///tmp/custom.db")
	os.Setenv("COZE_MQ_TYPE", "nsq")
	t.Cleanup(func() {
		os.Unsetenv("LITE_MODE")
		os.Unsetenv("DB_URI")
		os.Unsetenv("COZE_MQ_TYPE")
	})

	applyLitePreset()

	// Should preserve pre-set values
	assert.Equal(t, "sqlite:///tmp/custom.db", os.Getenv("DB_URI"))
	assert.Equal(t, "nsq", os.Getenv("COZE_MQ_TYPE"))
}
