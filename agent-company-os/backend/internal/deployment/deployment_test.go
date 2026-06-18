package deployment

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExecuteRunsCommandInsideAllowedRoot(t *testing.T) {
	root := t.TempDir()
	workdir := filepath.Join(root, "app")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatal(err)
	}
	svc := NewService(root, time.Second, 10000)
	res, err := svc.Execute(context.Background(), ProjectConfig{ProjectName: "app", Workdir: workdir, AllowedRoot: root, DeployCommand: []string{"echo", "deployed"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "completed" || res.Output != "deployed" {
		t.Fatalf("unexpected result %#v", res)
	}
}

func TestExecuteRejectsWorkdirOutsideAllowedRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	svc := NewService(root, time.Second, 10000)
	res, err := svc.Execute(context.Background(), ProjectConfig{ProjectName: "app", Workdir: outside, AllowedRoot: root, DeployCommand: []string{"echo", "deployed"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.ErrorClass != "invalid_workdir" || res.Status != "failed" {
		t.Fatalf("unexpected result %#v", res)
	}
}

func TestExecuteAllowsRelativeScriptInsideWorkdir(t *testing.T) {
	root := t.TempDir()
	workdir := filepath.Join(root, "app")
	if err := os.MkdirAll(filepath.Join(workdir, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(workdir, "scripts", "deploy.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho script-deployed\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	svc := NewService(root, time.Second, 10000)
	res, err := svc.Execute(context.Background(), ProjectConfig{ProjectName: "app", Workdir: workdir, AllowedRoot: root, DeployCommand: []string{"./scripts/deploy.sh"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "completed" || res.Output != "script-deployed" {
		t.Fatalf("unexpected result %#v", res)
	}
}

func TestExecuteRunsMultipleServiceCommands(t *testing.T) {
	root := t.TempDir()
	workdir := filepath.Join(root, "app")
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		t.Fatal(err)
	}
	svc := NewService(root, time.Second, 10000)
	res, err := svc.Execute(context.Background(), ProjectConfig{
		ProjectName: "app",
		Workdir:     workdir,
		AllowedRoot: root,
		Services: []Target{
			{Name: "api", DeployCommand: []string{"echo", "api-deployed"}},
			{Name: "worker", DeployCommand: []string{"echo", "worker-deployed"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "api:\napi-deployed\nworker:\nworker-deployed"
	if res.Status != "completed" || res.Output != want {
		t.Fatalf("unexpected result %#v", res)
	}
}
