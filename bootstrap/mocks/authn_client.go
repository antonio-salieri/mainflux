// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/mainflux/mainflux"
	"google.golang.org/grpc"
)

// AuthnClient authn client mock
type AuthnClient struct{}

var _ mainflux.AuthNServiceClient = &AuthnClient{}

// Issue method mock
func (ac *AuthnClient) Issue(ctx context.Context, in *mainflux.IssueReq, opts ...grpc.CallOption) (*mainflux.Token, error) {
	return &mainflux.Token{}, nil
}

// Identify method mock
func (ac *AuthnClient) Identify(ctx context.Context, in *mainflux.Token, opts ...grpc.CallOption) (*mainflux.UserID, error) {
	return &mainflux.UserID{}, nil
}
