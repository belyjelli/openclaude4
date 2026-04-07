package tools

import (
	"context"
	"testing"
)

func TestSubTaskDepth(t *testing.T) {
	ctx := context.Background()
	if SubTaskDepth(ctx) != 0 {
		t.Fatalf("want 0, got %d", SubTaskDepth(ctx))
	}
	ctx = WithSubTaskDepth(ctx, 1)
	if SubTaskDepth(ctx) != 1 {
		t.Fatalf("want 1, got %d", SubTaskDepth(ctx))
	}
}
