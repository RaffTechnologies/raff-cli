package config

import "testing"

func TestMaskSecret(t *testing.T) {
	if got := MaskSecret("raff_737e4c6f93aebadb373c1943271a6ad8"); got != "raff_737e…" {
		t.Errorf("got %q", got)
	}
	if got := MaskSecret("short"); got != "*****" {
		t.Errorf("got %q", got)
	}
}
