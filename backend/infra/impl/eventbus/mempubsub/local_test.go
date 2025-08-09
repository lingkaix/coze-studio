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

package mempubsub

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/coze-dev/coze-studio/backend/infra/contract/eventbus"
	"github.com/stretchr/testify/assert"
)

func TestLocalProducerConsumer(t *testing.T) {
	if os.Getenv("MEM_LOCAL_TEST") != "true" {
		return
	}

	topic := "test_topic"
	group := "test_group"

	// register consumer
	var wg sync.WaitGroup
	wg.Add(1)
	var got []byte
	err := RegisterConsumer("", topic, group, ConsumerHandlerFunc(func(ctx context.Context, msgBody []byte) error {
		got = append([]byte{}, msgBody...)
		wg.Done()
		return nil
	}))
	assert.NoError(t, err)

	// create producer
	p, err := NewProducer("", topic, group)
	assert.NoError(t, err)

	// send message
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	assert.NoError(t, p.Send(ctx, []byte("hello")))

	// wait for consume
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		assert.Equal(t, []byte("hello"), got)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

// ConsumerHandlerFunc adapts a function to the expected ConsumerHandler.
type ConsumerHandlerFunc func(ctx context.Context, msgBody []byte) error

func (f ConsumerHandlerFunc) HandleMessage(ctx context.Context, msg *eventbus.Message) error {
	return f(ctx, msg.Body)
}
