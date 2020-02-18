// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/internal/pkg/server"
	"github.com/mainflux/mainflux/logger"
)

const (
	stream = "mainflux.things"
	group  = "mainflux.bootstrap"

	thingPrefix     = "thing."
	thingRemove     = thingPrefix + "remove"
	thingDisconnect = thingPrefix + "disconnect"

	channelPrefix = "channel."
	channelUpdate = channelPrefix + "update"
	channelRemove = channelPrefix + "remove"

	exists = "BUSYGROUP Consumer Group name already exists"
)

// Subscriber represents event source for things and channels provisioning.
type Subscriber interface {
	server.GracefulServer

	// Subscribes to given subject and receives events.
	Subscribe(string) error
}

// RedisClient narrow interface of redis client
type RedisClient interface {
	XGroupCreateMkStream(stream, group, start string) *redis.StatusCmd
	XReadGroup(a *redis.XReadGroupArgs) *redis.XStreamSliceCmd
	XAck(stream, group string, ids ...string) *redis.IntCmd
}

type shutdownSignal uint8

const (
	req shutdownSignal = iota + 1
	ack
)

type eventStore struct {
	svc       bootstrap.Service
	client    RedisClient
	consumer  string
	logger    logger.Logger
	shutdownC chan shutdownSignal
}

// NewEventStore returns new event store instance.
func NewEventStore(svc bootstrap.Service, client RedisClient, consumer string, log logger.Logger) Subscriber {
	return eventStore{
		svc:       svc,
		client:    client,
		consumer:  consumer,
		logger:    log,
		shutdownC: make(chan shutdownSignal),
	}
}

func (es eventStore) Subscribe(subject string) error {
	err := es.client.XGroupCreateMkStream(stream, group, "$").Err()
	if err != nil && err.Error() != exists {
		return err
	}

	streamsChan := make(chan []redis.XStream)
	go func() {
		for {
			es.logger.Info("XReadGroup - reading")
			streams, err := es.client.XReadGroup(&redis.XReadGroupArgs{
				Group:    group,
				Consumer: es.consumer,
				Streams:  []string{stream, ">"},
				Count:    100,
			}).Result()
			es.logger.Info(fmt.Sprintf("XReadGroup - reading err: %+v", err))
			if err != nil || len(streams) == 0 {
				continue
			}
			es.logger.Info("XReadGroup - read")

			streamsChan <- streams
		}
	}()

	for {
		select {
		case shutdonwReq := <-es.shutdownC:
			if shutdonwReq != req {
				es.logger.Error(fmt.Sprintf("Received unexpected shutdown request: %+v. Continuing ...", req))
				continue
			}
			es.logger.Info("Event source listener received shutdown signal")
			es.shutdownC <- ack

			return nil
		case streams := <-streamsChan:
			for _, msg := range streams[0].Messages {
				event := msg.Values

				var err error
				switch event["operation"] {
				case thingRemove:
					rte := decodeRemoveThing(event)
					err = es.handleRemoveThing(rte)
				case thingDisconnect:
					dte := decodeDisconnectThing(event)
					err = es.handleDisconnectThing(dte)
				case channelUpdate:
					uce := decodeUpdateChannel(event)
					err = es.handleUpdateChannel(uce)
				case channelRemove:
					rce := decodeRemoveChannel(event)
					err = es.handleRemoveChannel(rce)
				}
				if err != nil {
					es.logger.Warn(fmt.Sprintf("Failed to handle event sourcing: %s", err.Error()))
					break
				}
				es.client.XAck(stream, group, msg.ID)
			}
		}
	}
}

// Stop gracefully stops event store listener
func (es eventStore) Stop(logger logger.Logger, timeout time.Duration) error {
	t := time.NewTimer(timeout)
	defer t.Stop()

	logger.Info("Stopping event source listener")
	es.shutdownC <- req

	select {
	case resp := <-es.shutdownC:
		if resp != ack {
			return fmt.Errorf("Event source listener received enexpected shutdown request response: %+v", resp)
		}
	case <-t.C:
		return fmt.Errorf("Event source listener graceful shutdown timeout - forced exit")
	}
	logger.Info("Event source listener stopped without errors")
	return nil
}

func decodeRemoveThing(event map[string]interface{}) removeEvent {
	return removeEvent{
		id: read(event, "id", ""),
	}
}

func decodeUpdateChannel(event map[string]interface{}) updateChannelEvent {
	strmeta := read(event, "metadata", "{}")
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(strmeta), metadata); err != nil {
		metadata = map[string]interface{}{}
	}

	return updateChannelEvent{
		id:       read(event, "id", ""),
		name:     read(event, "name", ""),
		metadata: metadata,
	}
}

func decodeRemoveChannel(event map[string]interface{}) removeEvent {
	return removeEvent{
		id: read(event, "id", ""),
	}
}

func decodeDisconnectThing(event map[string]interface{}) disconnectEvent {
	return disconnectEvent{
		channelID: read(event, "chan_id", ""),
		thingID:   read(event, "thing_id", ""),
	}
}

func (es eventStore) handleRemoveThing(rte removeEvent) error {
	return es.svc.RemoveConfigHandler(rte.id)
}

func (es eventStore) handleUpdateChannel(uce updateChannelEvent) error {
	channel := bootstrap.Channel{
		ID:       uce.id,
		Name:     uce.name,
		Metadata: uce.metadata,
	}
	return es.svc.UpdateChannelHandler(channel)
}

func (es eventStore) handleRemoveChannel(rce removeEvent) error {
	return es.svc.RemoveChannelHandler(rce.id)
}

func (es eventStore) handleDisconnectThing(dte disconnectEvent) error {
	return es.svc.DisconnectThingHandler(dte.channelID, dte.thingID)
}

func read(event map[string]interface{}, key, def string) string {
	val, ok := event[key].(string)
	if !ok {
		return def
	}

	return val
}
