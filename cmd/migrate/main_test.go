package main

import "testing"

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantAction string
		wantSteps  int
		wantError  bool
	}{
		{name: "up", args: []string{"up"}, wantAction: "up"},
		{name: "down defaults to one step", args: []string{"down"}, wantAction: "down", wantSteps: 1},
		{name: "down with steps", args: []string{"down", "3"}, wantAction: "down", wantSteps: 3},
		{name: "missing action", wantError: true},
		{name: "unknown action", args: []string{"reset"}, wantError: true},
		{name: "invalid steps", args: []string{"down", "0"}, wantError: true},
		{name: "too many arguments", args: []string{"up", "1"}, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseCommand(test.args)
			if test.wantError {
				if err == nil {
					t.Fatal("parseCommand() returned no error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCommand() returned an error: %v", err)
			}
			if got.action != test.wantAction || got.steps != test.wantSteps {
				t.Fatalf("parseCommand() = %+v, want action=%q steps=%d", got, test.wantAction, test.wantSteps)
			}
		})
	}
}
