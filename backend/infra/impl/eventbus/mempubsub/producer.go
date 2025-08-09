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
	"fmt"
	"sync"

	"gocloud.dev/pubsub"

	"github.com/coze-dev/coze-studio/backend/infra/contract/eventbus"
	"github.com/coze-dev/coze-studio/backend/pkg/lang/signal"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/safego"
	_ "gocloud.dev/pubsub/mempubsub"
)

var (
	topicRegistryMu sync.Mutex
	topicRegistry   = map[string]*pubsub.Topic{}
)

func getOrOpenTopic(ctx context.Context, topicName string) (*pubsub.Topic, error) {
	topicRegistryMu.Lock()
	defer topicRegistryMu.Unlock()
	if t, ok := topicRegistry[topicName]; ok {
		return t, nil
	}
	t, err := pubsub.OpenTopic(ctx, fmt.Sprintf("mem://%s", topicName))
	if err != nil {
		return nil, err
	}
	topicRegistry[topicName] = t
	return t, nil
}

type producerImpl struct {
	topicName string
	topic     *pubsub.Topic
}

// NewProducer creates a mempubsub producer. nameServer is ignored.
func NewProducer(_ string, topic, _ string) (eventbus.Producer, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic is empty")
	}
	ctx := context.Background()
	t, err := getOrOpenTopic(ctx, topic)
	if err != nil {
		return nil, err
	}

	p := &producerImpl{
		topicName: topic,
		topic:     t,
	}

	safego.Go(ctx, func() {
		signal.WaitExit()
		if err := p.topic.Shutdown(context.Background()); err != nil {
			logs.Errorf("[mempubsub] shutdown topic %s failed: %v", p.topicName, err)
		}
	})
	return p, nil
}

func (p *producerImpl) Send(ctx context.Context, body []byte, _ ...eventbus.SendOpt) error {
	return p.topic.Send(ctx, &pubsub.Message{Body: body})
}

func (p *producerImpl) BatchSend(ctx context.Context, bodyArr [][]byte, _ ...eventbus.SendOpt) error {
	for _, b := range bodyArr {
		if err := p.topic.Send(ctx, &pubsub.Message{Body: b}); err != nil {
			return err
		}
	}
	return nil
}
