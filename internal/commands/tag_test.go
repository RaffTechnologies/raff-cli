package commands

import (
	"testing"

	"github.com/rafftechnologies/raff-go"
)

func TestTagColumns(t *testing.T) {
	one, zero := 1, 0
	for _, c := range []struct {
		p    *int
		want string
	}{{nil, "-"}, {&zero, "-"}, {&one, "1"}} {
		if got := tagPriority(c.p); got != c.want {
			t.Errorf("tagPriority(%v) = %q, want %q", c.p, got, c.want)
		}
	}
	if got := tagFirewallNames(nil); got != "-" {
		t.Errorf("no firewalls: %q", got)
	}
	links := []raff.TagFirewallLink{{SecurityGroupName: "web-server"}, {SecurityGroupName: "db-only"}}
	if got := tagFirewallNames(links); got != "web-server, db-only" {
		t.Errorf("firewalls: %q", got)
	}
}
