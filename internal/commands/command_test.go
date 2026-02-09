package commands

import (
	"errors"
	"testing"
)

func TestParseSupportedCommands(t *testing.T) {
	cases := []struct {
		in       string
		typeWant Type
	}{
		{"/add pay rent tomorrow", TypeAdd},
		{"snooze overdue 2 days", TypeSnooze},
		{"show tasks tag:finance", TypeShow},
		{"reschedule selected next monday", TypeReschedule},
	}

	for _, tc := range cases {
		cmd, err := Parse(tc.in)
		if err != nil {
			t.Fatalf("parse %q failed: %v", tc.in, err)
		}
		if cmd.Type != tc.typeWant {
			t.Fatalf("parse %q type = %s, want %s", tc.in, cmd.Type, tc.typeWant)
		}
	}
}

func TestParseUnknownCommand(t *testing.T) {
	_, err := Parse("/unknown do x")
	if err == nil {
		t.Fatal("expected error")
	}
	var ce *CommandError
	if !errors.As(err, &ce) || ce.Code != ErrCodeUnknownCommand {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

func TestExecuteDispatch(t *testing.T) {
	cmd, err := Parse("/add write docs")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	called := false
	res, err := Execute(cmd, Handlers{
		Add: func(a AddArgs) (Result, error) {
			called = true
			if a.Title != "write docs" {
				t.Fatalf("unexpected title: %q", a.Title)
			}
			return Result{Message: "ok"}, nil
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if !called || res.Message != "ok" {
		t.Fatalf("dispatch failed, called=%v res=%+v", called, res)
	}
}

func TestExecuteMissingHandler(t *testing.T) {
	cmd, err := Parse("show tasks")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, err = Execute(cmd, Handlers{})
	if err == nil {
		t.Fatal("expected error")
	}
	var ce *CommandError
	if !errors.As(err, &ce) || ce.Code != ErrCodeHandlerMissing {
		t.Fatalf("expected missing handler error, got %v", err)
	}
}
