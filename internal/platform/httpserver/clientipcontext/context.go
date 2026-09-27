// Package clientipcontext stores a client IP resolved by HTTP middleware.
package clientipcontext

import "context"

type key struct{}

func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, key{}, ip)
}

func FromContext(ctx context.Context) string {
	ip, _ := ctx.Value(key{}).(string)
	return ip
}
