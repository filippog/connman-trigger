// SPDX-License-Identifier: Apache-2.0
package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestIsExecutable verifies that isExecutable correctly identifies executable files
func TestIsExecutable(t *testing.T) {
	file, err := ioutil.TempFile("", "exe_test")
	if err != nil {
		t.Fatalf("TempFile failed: %v", err)
	}
	name := file.Name()
	file.Close()
	defer os.Remove(name)

	if isExecutable(name) {
		t.Errorf("expected non-executable file to return false, got true")
	}

	if err := os.Chmod(name, 0755); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}

	if !isExecutable(name) {
		t.Errorf("expected executable file to return true, got false")
	}
}

// TestExecuteScriptsInDir verifies that executeScriptsInDir runs executable scripts
func TestExecuteScriptsInDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping script execution test on Windows")
	}

	dir, err := ioutil.TempDir("", "exec_test_dir")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}
	defer os.RemoveAll(dir)

	outFile := filepath.Join(dir, "out.txt")
	script := "#!/bin/sh\n\necho \"$1\" > \"$OUTFILE\"\n"
	scriptPath := filepath.Join(dir, "script.sh")
	if err := ioutil.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Chmod(scriptPath, 0755); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}

	executeScriptsInDir(dir, "up", map[string]string{"OUTFILE": outFile})

	data, err := ioutil.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	expected := "up\n"
	if string(data) != expected {
		t.Errorf("unexpected content in outFile: got %q, want %q", string(data), expected)
	}
}
