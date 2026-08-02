package main

import "testing"

func TestVetInvocation(t *testing.T) {
	for _, arguments := range [][]string{{"-flags"}, {"-c=3"}, {"package.cfg"}, {"-V=full"}} {
		if !vetInvocation(arguments) {
			t.Errorf("vetInvocation(%q) = false", arguments)
		}
	}
	if vetInvocation([]string{"check", "./..."}) {
		t.Fatal("ordinary CLI invocation detected as vet protocol")
	}
}
