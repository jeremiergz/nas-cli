package sliceutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Contains_Returns_True_When_Slice_Contains_Target(t *testing.T) {
	tests := map[any][]any{
		"a": {"a", "b", "c"},
		"b": {"a", "b", "c"},
		"c": {"a", "b", "c"},
		1:   {1, 2, 3},
		2:   {1, 2, 3},
		3:   {1, 2, 3},
	}

	for target, elems := range tests {
		if !Contains(elems, target) {
			assert.Fail(t, fmt.Sprintf("Expected slice to contain target \"%v\"", target))
		}
	}
}

func Test_Contains_Returns_False_When_Target_Is_Not_In_Slice(t *testing.T) {
	tests := map[any][]any{
		"d": {"a", "b", "c"},
		4:   {1, 2, 3},
	}

	for target, elems := range tests {
		if Contains(elems, target) {
			assert.Fail(t, fmt.Sprintf("Expected target \"%v\" to not be in slice", target))
		}
	}
}
