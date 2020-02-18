// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import mfsdk "github.com/mainflux/mainflux/sdk/go"

// SDK Mainflux sdk mock
type SDK struct{}

// CreateUser mock sdk method
func (s *SDK) CreateUser(user mfsdk.User) error {
	return nil
}

// CreateToken mock sdk method
func (s *SDK) CreateToken(user mfsdk.User) (string, error) {
	return "", nil
}

// CreateThing mock sdk method
func (s *SDK) CreateThing(thing mfsdk.Thing, token string) (string, error) {
	return "", nil
}

// CreateThings mock sdk method
func (s *SDK) CreateThings(things []mfsdk.Thing, token string) ([]mfsdk.Thing, error) {
	return []mfsdk.Thing{}, nil
}

// Things mock sdk method
func (s *SDK) Things(token string, offset, limit uint64, name string) (mfsdk.ThingsPage, error) {
	return mfsdk.ThingsPage{}, nil
}

// ThingsByChannel mock sdk method
func (s *SDK) ThingsByChannel(token, chanID string, offset, limit uint64) (mfsdk.ThingsPage, error) {
	return mfsdk.ThingsPage{}, nil
}

// Thing mock sdk method
func (s *SDK) Thing(id, token string) (mfsdk.Thing, error) {
	return mfsdk.Thing{}, nil
}

// UpdateThing mock sdk method
func (s *SDK) UpdateThing(thing mfsdk.Thing, token string) error {
	return nil
}

// DeleteThing mock sdk method
func (s *SDK) DeleteThing(id, token string) error {
	return nil
}

// ConnectThing mock sdk method
func (s *SDK) ConnectThing(thingID, chanID, token string) error {
	return nil
}

// Connect mock sdk method
func (s *SDK) Connect(conns mfsdk.ConnectionIDs, token string) error {
	return nil
}

// DisconnectThing mock sdk method
func (s *SDK) DisconnectThing(thingID, chanID, token string) error {
	return nil
}

// CreateChannel mock sdk method
func (s *SDK) CreateChannel(channel mfsdk.Channel, token string) (string, error) {
	return "", nil
}

// CreateChannels mock sdk method
func (s *SDK) CreateChannels(channels []mfsdk.Channel, token string) ([]mfsdk.Channel, error) {
	return []mfsdk.Channel{}, nil
}

// Channels mock sdk method
func (s *SDK) Channels(token string, offset, limit uint64, name string) (mfsdk.ChannelsPage, error) {
	return mfsdk.ChannelsPage{}, nil
}

// ChannelsByThing mock sdk method
func (s *SDK) ChannelsByThing(token, thingID string, offset, limit uint64) (mfsdk.ChannelsPage, error) {
	return mfsdk.ChannelsPage{}, nil
}

// Channel mock sdk method
func (s *SDK) Channel(id, token string) (mfsdk.Channel, error) {
	return mfsdk.Channel{}, nil
}

// UpdateChannel mock sdk method
func (s *SDK) UpdateChannel(channel mfsdk.Channel, token string) error {
	return nil
}

// DeleteChannel mock sdk method
func (s *SDK) DeleteChannel(id, token string) error {
	return nil
}

// SendMessage mock sdk method
func (s *SDK) SendMessage(chanID, msg, token string) error {
	return nil
}

// ReadMessages mock sdk method
func (s *SDK) ReadMessages(chanID, token string) (mfsdk.MessagesPage, error) {
	return mfsdk.MessagesPage{}, nil
}

// SetContentType mock sdk method
func (s *SDK) SetContentType(ct mfsdk.ContentType) error {
	return nil
}

// Version mock sdk method
func (s *SDK) Version() (string, error) {
	return "", nil
}
