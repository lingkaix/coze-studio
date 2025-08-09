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

	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"

	"github.com/coze-dev/coze-studio/backend/infra/contract/eventbus"
	"github.com/coze-dev/coze-studio/backend/pkg/logs"
	"github.com/coze-dev/coze-studio/backend/pkg/safego"
)

// RegisterConsumer subscribes to a mempubsub topic. nameServer and group are ignored.
func RegisterConsumer(_ string, topic, _ string, handler eventbus.ConsumerHandler, _ ...eventbus.ConsumerOpt) error {
	if topic == "" {
		return fmt.Errorf("topic is empty")
	}
	if handler == nil {
		return fmt.Errorf("consumer handler is nil")
	}
	ctx := context.Background()

    // Ensure topic exists before opening subscription
    if _, err := getOrOpenTopic(ctx, topic); err != nil {
        return err
    }
    sub, err := pubsub.OpenSubscription(ctx, fmt.Sprintf("mem://%s", topic))
	if err != nil {
		return err
	}

	safego.Go(ctx, func() {
		defer func() {
			if err := sub.Shutdown(context.Background()); err != nil {
				logs.Errorf("[mempubsub] shutdown subscription %s failed: %v", topic, err)
			}
		}()
		// block until exit, receiving sequentially
		for {
			msg, err := sub.Receive(ctx)
			if err != nil {
				return
			}
			if err := handler.HandleMessage(ctx, &eventbus.Message{Topic: topic, Body: msg.Body}); err != nil {
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	})
	return nil
}
