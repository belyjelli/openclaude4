package ocrpc

import (
	"testing"

	"github.com/gitlawb/openclaude4/internal/grpc/openclaudev4"
)

func TestUserInputToPermissionOutcome(t *testing.T) {
	y := "y"
	o := userInputToPermissionOutcome(&openclaudev4.UserInput{Reply: y})
	if !o.Allow || o.EnableSessionAutoApprove || len(o.AddAllowRules) > 0 {
		t.Fatalf("y: %+v", o)
	}
	auto := true
	rule := "Bash(git:*)"
	o2 := userInputToPermissionOutcome(&openclaudev4.UserInput{
		Reply:                    "yes",
		EnableSessionAutoApprove: &auto,
		AddAllowRule:             &rule,
	})
	if !o2.Allow || !o2.EnableSessionAutoApprove || len(o2.AddAllowRules) != 1 {
		t.Fatalf("structured yes: %+v", o2)
	}
	fb := "stop"
	o3 := userInputToPermissionOutcome(&openclaudev4.UserInput{Reply: "n", DeclineFeedback: &fb})
	if o3.Allow || o3.DeclineUserNote != "stop" {
		t.Fatalf("deny: %+v", o3)
	}
}
