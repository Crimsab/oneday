package main

import "testing"

func TestWantsVersion(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{args: nil, want: false},
		{args: []string{"--version"}, want: true},
		{args: []string{"version"}, want: true},
		{args: []string{"play"}, want: false},
	}

	for _, tc := range tests {
		if got := wantsVersion(tc.args); got != tc.want {
			t.Fatalf("wantsVersion(%v) = %v, want %v", tc.args, got, tc.want)
		}
	}
}
