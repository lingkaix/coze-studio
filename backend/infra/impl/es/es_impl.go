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

package es

import (
	"fmt"
	"os"

	"github.com/coze-dev/coze-studio/backend/infra/contract/es"
	"github.com/coze-dev/coze-studio/backend/infra/impl/es/duckdb"
	"github.com/coze-dev/coze-studio/backend/infra/impl/es/noop"
)

type (
	Client          = es.Client
	Types           = es.Types
	BulkIndexer     = es.BulkIndexer
	BulkIndexerItem = es.BulkIndexerItem
	BoolQuery       = es.BoolQuery
	Query           = es.Query
	Response        = es.Response
	Request         = es.Request
)

func New() (Client, error) {
	v := os.Getenv("ES_VERSION")
	// Prefer explicit SEARCH_URI, regardless of lite mode
	if uri := os.Getenv("SEARCH_URI"); uri != "" && len(uri) >= 9 && uri[:9] == "duckdb://" {
		return duckdb.New(uri[9:]), nil
	}
	if os.Getenv("LITE_MODE") == "1" || os.Getenv("COZE_LITE") == "1" {
		return noop.New(), nil
	}
	if v == "v8" {
		return newES8()
	} else if v == "v7" {
		return newES7()
	}

	return nil, fmt.Errorf("unsupported es version %s", v)
}
