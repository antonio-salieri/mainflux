// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import "github.com/go-redis/redis"

// RedisClient redis client mock
type RedisClient struct{}

// XGroupCreateMkStream mocked XGroupCreateMkStream redis client method
func (rc *RedisClient) XGroupCreateMkStream(stream, group, start string) *redis.StatusCmd {
	return &redis.StatusCmd{}
}

// XReadGroup mocked XReadGroup redis client method
func (rc *RedisClient) XReadGroup(a *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	return &redis.XStreamSliceCmd{}
}

// XAck mocked XAck redis client method
func (rc *RedisClient) XAck(stream, group string, ids ...string) *redis.IntCmd {
	return &redis.IntCmd{}
}
