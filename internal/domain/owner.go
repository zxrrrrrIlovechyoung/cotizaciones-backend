package domain

import "context"

type ownerKey struct{}

func WithOwner(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, ownerKey{}, id)
}
func Owner(ctx context.Context) int64 { id, _ := ctx.Value(ownerKey{}).(int64); return id }
