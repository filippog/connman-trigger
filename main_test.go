// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/godbus/dbus/v5"
)

// TestIsExecutable verifies that isExecutable correctly identifies executable files
func TestIsExecutable(t *testing.T) {
	file, err := os.CreateTemp("", "exe_test")
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

// Fake implementations for testing getNetworkDetailsForService
// fakeBusObject implements dbus.BusObject for tests
type fakeBusObject struct {
	call *dbus.Call
}

func (fbo *fakeBusObject) Call(method string, flags dbus.Flags, args ...interface{}) *dbus.Call {
	return fbo.call
}

// CallWithContext satisfies dbus.BusObject
func (fbo *fakeBusObject) CallWithContext(ctx context.Context, method string, flags dbus.Flags, args ...interface{}) *dbus.Call {
	return fbo.call
}

// Go satisfies dbus.BusObject
func (fbo *fakeBusObject) Go(method string, flags dbus.Flags, ch chan *dbus.Call, args ...interface{}) *dbus.Call {
	return fbo.call
}

// GoWithContext satisfies dbus.BusObject
func (fbo *fakeBusObject) GoWithContext(ctx context.Context, method string, flags dbus.Flags, ch chan *dbus.Call, args ...interface{}) *dbus.Call {
	return fbo.call
}

// AddMatchSignal satisfies dbus.BusObject
func (fbo *fakeBusObject) AddMatchSignal(iface, member string, options ...dbus.MatchOption) *dbus.Call {
	return fbo.call
}

// RemoveMatchSignal satisfies dbus.BusObject
func (fbo *fakeBusObject) RemoveMatchSignal(iface, member string, options ...dbus.MatchOption) *dbus.Call {
	return fbo.call
}

// GetProperty satisfies dbus.BusObject
func (fbo *fakeBusObject) GetProperty(p string) (dbus.Variant, error) {
	return dbus.MakeVariant(nil), nil
}

// StoreProperty satisfies dbus.BusObject
func (fbo *fakeBusObject) StoreProperty(p string, value interface{}) error {
	return nil
}

// SetProperty satisfies dbus.BusObject
func (fbo *fakeBusObject) SetProperty(p string, v interface{}) error {
	return nil
}

// Destination satisfies dbus.BusObject
func (fbo *fakeBusObject) Destination() string {
	return ""
}

// Path satisfies dbus.BusObject
func (fbo *fakeBusObject) Path() dbus.ObjectPath {
	return ""
}

// fakeConn implements dbusConnection
type fakeConn struct {
	obj dbus.BusObject
}

func (fc fakeConn) Object(dest string, path dbus.ObjectPath) dbus.BusObject {
	return fc.obj
}

func TestGetNetworkDetailsForService(t *testing.T) {
	cases := []struct {
		name     string
		call     *dbus.Call
		wantName string
		wantType string
	}{
		{
			name: "GoodProperties",
			call: &dbus.Call{
				Body: []interface{}{map[string]dbus.Variant{
					"Name": dbus.MakeVariant("myssid"),
					"Type": dbus.MakeVariant("wifi"),
				}},
				Err: nil,
			},
			wantName: "myssid",
			wantType: "wifi",
		},
		{
			name: "CallError",
			call: &dbus.Call{
				Body: []interface{}{map[string]dbus.Variant{
					"Name": dbus.MakeVariant("name"),
					"Type": dbus.MakeVariant("type"),
				}},
				Err: errors.New("fail"),
			},
			wantName: "unknown",
			wantType: "unknown",
		},
		{
			name: "EmptyBody",
			call: &dbus.Call{
				Body: []interface{}{},
				Err:  nil,
			},
			wantName: "unknown",
			wantType: "unknown",
		},
		{
			name: "BadBodyType",
			call: &dbus.Call{
				Body: []interface{}{"notamap"},
				Err:  nil,
			},
			wantName: "unknown",
			wantType: "unknown",
		},
		{
			name: "MissingName",
			call: &dbus.Call{
				Body: []interface{}{map[string]dbus.Variant{
					"Type": dbus.MakeVariant("ethernet"),
				}},
				Err: nil,
			},
			wantName: "unknown",
			wantType: "unknown",
		},
		{
			name: "MissingType",
			call: &dbus.Call{
				Body: []interface{}{map[string]dbus.Variant{
					"Name": dbus.MakeVariant("ssidonly"),
				}},
				Err: nil,
			},
			wantName: "unknown",
			wantType: "unknown",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fbo := &fakeBusObject{call: tc.call}
			conn := fakeConn{obj: fbo}
			name, typ := getNetworkDetailsForService(conn, dbus.ObjectPath("/test"))
			if name != tc.wantName || typ != tc.wantType {
				t.Errorf("%s: got (%q, %q), want (%q, %q)", tc.name, name, typ, tc.wantName, tc.wantType)
			}
		})
	}
}

// TestExecuteScriptsInDir verifies that executeScriptsInDir runs executable scripts
func TestExecuteScriptsInDir(t *testing.T) {
	dir, err := os.MkdirTemp("", "exec_test_dir")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}
	defer os.RemoveAll(dir)

	outFile := filepath.Join(dir, "out.txt")
	script := "#!/bin/sh\n\necho \"$1\" > \"$OUTFILE\"\n"
	scriptPath := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.Chmod(scriptPath, 0755); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}

	executeScriptsInDir(dir, "up", map[string]string{"OUTFILE": outFile})

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	expected := "up\n"
	if string(data) != expected {
		t.Errorf("unexpected content in outFile: got %q, want %q", string(data), expected)
	}
}
