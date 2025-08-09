APP_NAME=coze-server

.PHONY: build-lite run-lite test-lite

build-lite:
	@echo "Building $(APP_NAME) (lite mode at runtime via env)..."
	GOFLAGS="-gcflags=all=-N -gcflags=all=-l" go build -o bin/$(APP_NAME) ./

run-lite: build-lite
	@mkdir -p ./var/data/sqlite ./var/data/blob ./var/data/badger ./var/data/duckdb
	LITE_MODE=1 COZE_LITE=1 \
	DB_URI=sqlite://./var/data/sqlite/app.db \
	SEARCH_URI=duckdb://./var/data/duckdb/lite.duckdb \
	COZE_MQ_TYPE=mem \
	STORAGE_TYPE=file BLOB_DIR=./var/data/blob \
	CACHE_URI=mem:// \
	./bin/$(APP_NAME)

test-lite:
	@echo "Running lite-related tests"
	go test ./infra/impl/sqlite -gcflags=all=-N -gcflags=all=-l
	go test ./infra/impl/storage/fileblob -gcflags=all=-N -gcflags=all=-l
	go test ./application/base/appinfra -run TestApplyLitePreset_ -gcflags=all=-N -gcflags=all=-l
	go test ./infra/impl/es/duckdb -gcflags=all=-N -gcflags=all=-l
	go test ./infra/impl/cache/badger -gcflags=all=-N -gcflags=all=-l

.PHONY: clean-lite-test-data
clean-lite-test-data:
	@echo "Cleaning lite test data dirs"
	rm -rf ./var/data/duckdb ./var/data/badger || true


