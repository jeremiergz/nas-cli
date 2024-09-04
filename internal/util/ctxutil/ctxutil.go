package ctxutil

import (
	"context"
)

type ctxKey[T any] struct{}

// Attaches value to the context using the type "T" as key.
//
// This actually creates a singleton instance of "T" in the context.
func WithSingleton[T any](ctx context.Context, value T) context.Context {
	return context.WithValue(ctx, ctxKey[T]{}, value)
}

// Return the value "T" attached to the context.
//
// If there is no value attached, the zero value is returned.
func Singleton[T any](ctx context.Context) T {
	value, _ := ctx.Value(ctxKey[T]{}).(T)
	return value
}
