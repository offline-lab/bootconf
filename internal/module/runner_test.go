package module

import (
	"context"
	"testing"
)

type mockModule struct {
	name    string
	success bool
	msg     string
}

func (mock *mockModule) Name() string { return mock.name }
func (mock *mockModule) Run(_ context.Context, _ bool) Result {
	return Result{Section: mock.name, Success: mock.success, Message: mock.msg}
}

func TestRunnerAllSuccess(t *testing.T) {
	mods := []Module{
		&mockModule{name: "system", success: true, msg: "ok"},
		&mockModule{name: "ssh", success: true, msg: "ok"},
	}

	runner := NewRunner(mods)
	results := runner.Run(context.Background(), false, "")

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, result := range results {
		if !result.Success {
			t.Errorf("section %s failed", result.Section)
		}
	}
}

func TestRunnerPartialFailure(t *testing.T) {
	mods := []Module{
		&mockModule{name: "system", success: true, msg: "ok"},
		&mockModule{name: "wifi", success: false, msg: ""},
		&mockModule{name: "ssh", success: true, msg: "ok"},
	}

	runner := NewRunner(mods)
	results := runner.Run(context.Background(), false, "")

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	wifiFailed := false
	for _, result := range results {
		if result.Section == "wifi" && !result.Success {
			wifiFailed = true
		}
	}
	if !wifiFailed {
		t.Error("wifi should have failed")
	}
}

func TestRunnerSingleSection(t *testing.T) {
	mods := []Module{
		&mockModule{name: "system", success: true},
		&mockModule{name: "ssh", success: true},
	}

	runner := NewRunner(mods)
	results := runner.Run(context.Background(), false, "ssh")

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Section != "ssh" {
		t.Errorf("expected ssh, got %s", results[0].Section)
	}
}

func TestRunnerEmpty(t *testing.T) {
	runner := NewRunner(nil)
	results := runner.Run(context.Background(), false, "")

	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}
