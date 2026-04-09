package tools

import "testing"

func TestIsDockerSafeReadOnlyCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{`docker ps`, true},
		{`docker ps -a`, true},
		{`docker images`, true},
		{`/usr/local/bin/docker images`, true},
		{`docker logs -f --tail 10 mycontainer`, true},
		{`docker logs mycontainer`, true},
		{`docker inspect nginx`, true},
		{`docker inspect -f "{{.Id}}" nginx`, true},
		{`docker run hello`, false},
		{`docker compose ps`, false},
		{`docker ps && id`, false},
		{`docker ps $IMAGE`, false},
		{`docker logs --unknown-flag x`, false},
	}
	for _, tt := range tests {
		if got := IsDockerSafeReadOnlyCommand(tt.cmd); got != tt.want {
			t.Errorf("IsDockerSafeReadOnlyCommand(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}

func TestIsBashReadOnlyNoConfirm_IncludesDocker(t *testing.T) {
	if !IsBashReadOnlyNoConfirm(`docker ps -a`) {
		t.Fatal("expected docker read-only to skip confirm")
	}
	if !IsBashReadOnlyNoConfirm(`docker logs --tail 5 svc`) {
		t.Fatal("expected docker logs read-only to skip confirm")
	}
	if IsBashReadOnlyNoConfirm(`docker run -it alpine sh`) {
		t.Fatal("expected mutating docker to require confirm")
	}
}
