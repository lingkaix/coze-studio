package noop

import (
	"context"

	contract "github.com/coze-dev/coze-studio/backend/infra/contract/es"
)

type client struct{}

func New() contract.Client { return &client{} }

func (c *client) Create(ctx context.Context, index, id string, document any) error { return nil }
func (c *client) Update(ctx context.Context, index, id string, document any) error { return nil }
func (c *client) Delete(ctx context.Context, index, id string) error               { return nil }
func (c *client) Search(ctx context.Context, index string, req *contract.Request) (*contract.Response, error) {
	return &contract.Response{Hits: contract.HitsMetadata{Hits: []contract.Hit{}}}, nil
}
func (c *client) Exists(ctx context.Context, index string) (bool, error) { return true, nil }
func (c *client) CreateIndex(ctx context.Context, index string, properties map[string]any) error {
	return nil
}
func (c *client) DeleteIndex(ctx context.Context, index string) error       { return nil }
func (c *client) Types() contract.Types                                     { return &types{} }
func (c *client) NewBulkIndexer(index string) (contract.BulkIndexer, error) { return &bulk{}, nil }

type types struct{}

func (t *types) NewLongNumberProperty() any         { return struct{}{} }
func (t *types) NewTextProperty() any               { return struct{}{} }
func (t *types) NewUnsignedLongNumberProperty() any { return struct{}{} }

type bulk struct{}

func (b *bulk) Add(ctx context.Context, item contract.BulkIndexerItem) error { return nil }
func (b *bulk) Close(ctx context.Context) error                              { return nil }
