// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package action

import (
	"context"
	"fmt"

	"github.com/elastic/fleet-server/v7/internal/pkg/bulk"
	"github.com/elastic/fleet-server/v7/internal/pkg/dl"

	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog/log"
)

const cacheSize = 5000

type TokenResolver struct {
	bulker bulk.Bulk
	cache  *lru.Cache
}

func NewTokenResolver(bulker bulk.Bulk) (*TokenResolver, error) {
	cache, err := lru.New(cacheSize)
	if err != nil {
		return nil, err
	}

	return &TokenResolver{
		bulker: bulker,
		cache:  cache,
	}, nil
}

func (r *TokenResolver) Resolve(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, dl.ErrNotFound
	}
	if v, ok := r.cache.Get(token); ok {
		seqno, ok := v.(int64)
		if !ok {
			return seqno, fmt.Errorf("unable to cast %v as type int64, detected type is: %T", v, v)
		}
		log.Debug().Str("token", token).Int64("seqno", seqno).Msg("Found token cached")
		return seqno, nil
	}

	seqno, err := dl.FindSeqNoByDocID(ctx, r.bulker, dl.QuerySeqNoByDocID, dl.FleetActions, token)
	if err != nil {
		return seqno, err
	}

	r.cache.Add(token, seqno)

	return seqno, nil
}
