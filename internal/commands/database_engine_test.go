package commands

import (
	"testing"

	"github.com/rafftechnologies/raff-go"
)

func TestDBEnginePrefersServerVersion(t *testing.T) {
	pg := raff.DatabaseEngine("postgres")
	major, full := "18", "18.6"
	cases := []struct {
		name string
		db   raff.Database
		want string
	}{
		{"full version once read", raff.Database{Engine: &pg, EngineVersion: &major, ServerVersion: &full}, "postgres 18.6"},
		{"major until the first reading", raff.Database{Engine: &pg, EngineVersion: &major}, "postgres 18"},
		{"no engine", raff.Database{EngineVersion: &major}, ""},
	}
	for _, c := range cases {
		if got := dbEngine(&c.db); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
