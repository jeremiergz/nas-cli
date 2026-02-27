package ctxutil

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
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

// Attaches a pterm spinner to the context and starts it with the given message.
func WithLoader(ctx context.Context, message string) (context.Context, error) {
	spinner, err := pterm.DefaultSpinner.Start(message)
	if err != nil {
		return nil, fmt.Errorf("could not start spinner: %w", err)
	}
	return context.WithValue(ctx, "spinner", spinner), nil
}

// Retrieves the pterm spinner attached to the context, if any.
func Loader(ctx context.Context) *pterm.SpinnerPrinter {
	if spinner, ok := ctx.Value("spinner").(*pterm.SpinnerPrinter); ok {
		return spinner
	}
	return nil
}
