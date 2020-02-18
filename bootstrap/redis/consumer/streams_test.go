// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumer_test

import (
	"testing"
	"time"

	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/bootstrap/mocks"
	"github.com/mainflux/mainflux/bootstrap/redis/consumer"
	log "github.com/mainflux/mainflux/logger/mocks"
)

func Test_eventStore_Stop(t *testing.T) {
	type fields struct {
		svc      bootstrap.Service
		client   *mocks.RedisClient
		consumer string
	}
	type args struct {
		logger  log.Logger
		timeout time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{{
		name: "Test listener stops gracefully",
		fields: fields{
			svc:      bootstrap.New(&mocks.AuthnClient{}, mocks.NewConfigsRepository(make(map[string]string)), &mocks.SDK{}, []byte("enc-key")),
			client:   &mocks.RedisClient{},
			consumer: "bootstrap_test_consumer",
		},
		args: args{
			logger:  log.Logger{},
			timeout: time.Second * 30,
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			es := consumer.NewEventStore(tt.fields.svc, tt.fields.client, tt.fields.consumer, log.Logger{})
			go es.Subscribe("mainflux.things")
			if err := es.Stop(tt.args.logger, tt.args.timeout); (err != nil) != tt.wantErr {
				t.Errorf("eventStore.Stop() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
