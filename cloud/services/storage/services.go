/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package storage

import (
	"context"

	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
)

type Service struct {
	scope *scope.ClusterScope
	ctx   context.Context
}

func NewService(ctx context.Context, scope *scope.ClusterScope) *Service {
	return &Service{
		scope: scope,
		ctx:   ctx,
	}
}
