/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package appinfra

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/coze-dev/coze-studio/backend/infra/contract/cache"
	"github.com/coze-dev/coze-studio/backend/infra/contract/coderunner"
	"github.com/coze-dev/coze-studio/backend/infra/contract/imagex"
	"github.com/coze-dev/coze-studio/backend/infra/contract/modelmgr"
	"github.com/coze-dev/coze-studio/backend/infra/impl/cache/redis"
	"github.com/coze-dev/coze-studio/backend/infra/impl/coderunner/direct"
	"github.com/coze-dev/coze-studio/backend/infra/impl/coderunner/sandbox"
	"github.com/coze-dev/coze-studio/backend/infra/impl/es"
	"github.com/coze-dev/coze-studio/backend/infra/impl/eventbus"
	"github.com/coze-dev/coze-studio/backend/infra/impl/idgen"
	"github.com/coze-dev/coze-studio/backend/infra/impl/imagex/veimagex"
	"github.com/coze-dev/coze-studio/backend/infra/impl/mysql"
	sqliteimpl "github.com/coze-dev/coze-studio/backend/infra/impl/sqlite"
	"github.com/coze-dev/coze-studio/backend/infra/impl/storage"
	fileimagex "github.com/coze-dev/coze-studio/backend/infra/impl/storage/fileimagex"
	"github.com/coze-dev/coze-studio/backend/types/consts"
)

type AppDependencies struct {
	DB                    *gorm.DB
	CacheCli              cache.Cmdable
	IDGenSVC              idgen.IDGenerator
	ESClient              es.Client
	ImageXClient          imagex.ImageX
	TOSClient             storage.Storage
	ResourceEventProducer eventbus.Producer
	AppEventProducer      eventbus.Producer
	ModelMgr              modelmgr.Manager
	CodeRunner            coderunner.Runner
}

func Init(ctx context.Context) (*AppDependencies, error) {
	deps := &AppDependencies{}
	var err error

	applyLitePreset()

	deps.DB, err = openDBFromEnv()
	if err != nil {
		return nil, err
	}

	deps.CacheCli = redis.New()

	deps.IDGenSVC, err = idgen.New(deps.CacheCli)
	if err != nil {
		return nil, err
	}

	deps.ESClient, err = es.New()
	if err != nil {
		return nil, err
	}

	deps.ImageXClient, err = initImageX(ctx)
	if err != nil {
		return nil, err
	}

	deps.TOSClient, err = initTOS(ctx)
	if err != nil {
		return nil, err
	}

	deps.ResourceEventProducer, err = initResourceEventBusProducer()
	if err != nil {
		return nil, err
	}

	deps.AppEventProducer, err = initAppEventProducer()
	if err != nil {
		return nil, err
	}

	deps.ModelMgr, err = initModelMgr()
	if err != nil {
		return nil, err
	}

	deps.CodeRunner = initCodeRunner()

	return deps, nil
}

func initImageX(ctx context.Context) (imagex.ImageX, error) {
	// In lite mode with local file storage, provide a file-backed imagex stub
	if os.Getenv(consts.StorageType) == "file" || os.Getenv("BLOB_URI") != "" {
		return fileimagex.New(os.Getenv("BLOB_DIR"))
	}

	uploadComponentType := os.Getenv(consts.FileUploadComponentType)

	if uploadComponentType != consts.FileUploadComponentTypeImagex {
		return storage.NewImagex(ctx)
	}
	return veimagex.New(
		os.Getenv(consts.VeImageXAK),
		os.Getenv(consts.VeImageXSK),
		os.Getenv(consts.VeImageXDomain),
		os.Getenv(consts.VeImageXUploadHost),
		os.Getenv(consts.VeImageXTemplate),
		[]string{os.Getenv(consts.VeImageXServerID)},
	)
}

func initTOS(ctx context.Context) (storage.Storage, error) {
	return storage.New(ctx)
}

func initResourceEventBusProducer() (eventbus.Producer, error) {
	nameServer := os.Getenv(consts.MQServer)
	resourceEventBusProducer, err := eventbus.NewProducer(nameServer,
		consts.RMQTopicResource, consts.RMQConsumeGroupResource, 1)
	if err != nil {
		return nil, fmt.Errorf("init resource producer failed, err=%w", err)
	}

	return resourceEventBusProducer, nil
}

func initAppEventProducer() (eventbus.Producer, error) {
	nameServer := os.Getenv(consts.MQServer)
	appEventProducer, err := eventbus.NewProducer(nameServer, consts.RMQTopicApp, consts.RMQConsumeGroupApp, 1)
	if err != nil {
		return nil, fmt.Errorf("init app producer failed, err=%w", err)
	}

	return appEventProducer, nil
}

func initCodeRunner() coderunner.Runner {
	switch typ := os.Getenv(consts.CodeRunnerType); typ {
	case "sandbox":
		getAndSplit := func(key string) []string {
			v := os.Getenv(key)
			if v == "" {
				return nil
			}
			return strings.Split(v, ",")
		}
		config := &sandbox.Config{
			AllowEnv:       getAndSplit(consts.CodeRunnerAllowEnv),
			AllowRead:      getAndSplit(consts.CodeRunnerAllowRead),
			AllowWrite:     getAndSplit(consts.CodeRunnerAllowWrite),
			AllowNet:       getAndSplit(consts.CodeRunnerAllowNet),
			AllowRun:       getAndSplit(consts.CodeRunnerAllowRun),
			AllowFFI:       getAndSplit(consts.CodeRunnerAllowFFI),
			NodeModulesDir: os.Getenv(consts.CodeRunnerNodeModulesDir),
			TimeoutSeconds: 0,
			MemoryLimitMB:  0,
		}
		if f, err := strconv.ParseFloat(os.Getenv(consts.CodeRunnerTimeoutSeconds), 64); err == nil {
			config.TimeoutSeconds = f
		} else {
			config.TimeoutSeconds = 60.0
		}
		if mem, err := strconv.ParseInt(os.Getenv(consts.CodeRunnerMemoryLimitMB), 10, 64); err == nil {
			config.MemoryLimitMB = mem
		} else {
			config.MemoryLimitMB = 100
		}
		return sandbox.NewRunner(config)
	default:
		return direct.NewRunner()
	}
}

// openDBFromEnv selects and opens the database based on DB_URI.
// Supported schemes:
// - sqlite://<path> → uses embedded SQLite provider
// - (default) → MySQL via MYSQL_DSN using existing mysql.New()
func openDBFromEnv() (*gorm.DB, error) {
	raw := os.Getenv("DB_URI")
	if raw == "" {
		return mysql.New()
	}

	// Accept simple prefix match to avoid fragile URL parsing for relative paths
	if strings.HasPrefix(strings.ToLower(raw), "sqlite://") {
		pathPart := raw[len("sqlite://"):]
		// Ensure directory exists for file-backed DB
		if dir := filepath.Dir(pathPart); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create sqlite data dir: %w", err)
			}
		}
		dsn := fmt.Sprintf("file:%s?mode=rwc&cache=shared", pathPart)
		return sqliteimpl.NewWithDSN(dsn)
	}

	// Fallback: support mysql:// DSN by translating into MYSQL_DSN for mysql.New(), or if MYSQL_DSN already set, just use it
	if strings.HasPrefix(strings.ToLower(raw), "mysql://") {
		// Convert URI to DSN expected by gorm mysql driver
		if u, err := url.Parse(raw); err == nil {
			// user:pass@tcp(host:port)/db?query
			user := u.User.Username()
			pass, _ := u.User.Password()
			host := u.Host
			dbname := strings.TrimPrefix(u.Path, "/")
			q := u.RawQuery
			dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s", user, pass, host, dbname)
			if q != "" {
				dsn = dsn + "?" + q
			}
			_ = os.Setenv("MYSQL_DSN", dsn)
		}
		return mysql.New()
	}

	// Unknown scheme → default to mysql
	return mysql.New()
}

// applyLitePreset enables embedded infra defaults when LITE_MODE=1 or COZE_LITE=1.
// It does not override explicitly provided envs.
func applyLitePreset() {
	lite := strings.TrimSpace(os.Getenv("LITE_MODE"))
	if lite == "" {
		lite = strings.TrimSpace(os.Getenv("COZE_LITE"))
	}
	if lite != "1" && strings.ToLower(lite) != "true" {
		return
	}

	// DB default → sqlite file under ./var/data/sqlite/app.db
	if os.Getenv("DB_URI") == "" {
		_ = os.MkdirAll("./var/data/sqlite", 0o755)
		_ = os.Setenv("DB_URI", "sqlite://./var/data/sqlite/app.db")
	}

	// MQ default → mempubsub
	if os.Getenv(consts.MQTypeKey) == "" {
		_ = os.Setenv(consts.MQTypeKey, "mem")
	}

	// Storage default → local file backend
	if os.Getenv(consts.StorageType) == "" {
		_ = os.Setenv(consts.StorageType, "file")
	}
	if os.Getenv("BLOB_DIR") == "" {
		_ = os.MkdirAll("./var/data/blob", 0o755)
		_ = os.Setenv("BLOB_DIR", "./var/data/blob")
	}

	// KV default → Badger path for embedded key-value store
	if os.Getenv("KV_URI") == "" && os.Getenv("CACHE_URI") == "" {
		_ = os.MkdirAll("./var/data/badger", 0o755)
		_ = os.Setenv("KV_URI", "badger://./var/data/badger")
	}

	// Search default → duckdb file under ./var/data/duckdb/lite.duckdb
	if os.Getenv("SEARCH_URI") == "" {
		_ = os.MkdirAll("./var/data/duckdb", 0o755)
		_ = os.Setenv("SEARCH_URI", "duckdb://./var/data/duckdb/lite.duckdb")
	}
}
