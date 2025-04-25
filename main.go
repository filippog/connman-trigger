// SPDX-License-Identifier: Apache-2.0
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/godbus/dbus/v5"
)

// dbusConnection abstracts D-Bus connection for testability
type dbusConnection interface {
	Object(dest string, path dbus.ObjectPath) dbus.BusObject
}

// Logger for the application
var logger = log.New(os.Stdout, "", log.LstdFlags)

// isExecutable checks if the file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode.IsRegular() && mode&0111 != 0
}

// executeScriptsInDir executes all executable scripts in a directory
func executeScriptsInDir(dir string, state string, envVars map[string]string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		logger.Printf("Error reading directory %s: %v\n", dir, err)
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {
		scriptPath := filepath.Join(dir, file.Name())
		if isExecutable(scriptPath) {
			logger.Printf("Executing: %s %s\n", scriptPath, state)
			cmd := exec.Command(scriptPath, state)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			// Set environment variables
			env := os.Environ()
			for key, value := range envVars {
				env = append(env, fmt.Sprintf("%s=%s", key, value))
			}
			cmd.Env = env

			if err := cmd.Run(); err != nil {
				logger.Printf("Error executing script %s: %v\n", scriptPath, err)
			}
		} else {
			logger.Printf("%s is not an executable file\n", scriptPath)
		}
	}
}

// executeScripts executes scripts in the given paths
func executeScripts(paths []string, state string, envVars map[string]string) {
	for _, path := range paths {
		if stat, err := os.Stat(path); err == nil && stat.IsDir() {
			executeScriptsInDir(path, state, envVars)
		}
	}
}

// getNetworkDetailsForService fetches SSID and connection type directly
// from the specified ConnMan service path.
func getNetworkDetailsForService(conn dbusConnection, servicePath dbus.ObjectPath) (string, string) {
	// Use Service.GetProperties to retrieve all properties
	service := conn.Object("net.connman", servicePath)
	propCall := service.Call("net.connman.Service.GetProperties", 0)
	if propCall.Err != nil || len(propCall.Body) < 1 {
		logger.Printf("Error calling Service.GetProperties on %s: %v", servicePath, propCall.Err)
		return "unknown", "unknown"
	}
	props, ok := propCall.Body[0].(map[string]dbus.Variant)
	if !ok {
		logger.Printf("Unexpected Service.GetProperties return type for %s: %T", servicePath, propCall.Body[0])
		return "unknown", "unknown"
	}
	// Extract Name and Type properties
	nameVar, okName := props["Name"]
	typeVar, okType := props["Type"]
	if !okName || !okType {
		return "unknown", "unknown"
	}
	name, _ := nameVar.Value().(string)
	ctype, _ := typeVar.Value().(string)
	return name, ctype
}

// https://git.kernel.org/pub/scm/network/connman/connman.git/tree/doc/overview-api.txt
// listenForNetworkChanges listens for network state changes over dbus
func listenForNetworkChanges(paths []string) {
	// Connect to the system bus
	conn, err := dbus.SystemBus()
	if err != nil {
		logger.Fatalf("Failed to connect to system bus: %v", err)
	}

	// https://git.kernel.org/pub/scm/network/connman/connman.git/tree/doc/service-api.txt
	// Subscribe to PropertyChanged signals from net.connman.Service
	if err := conn.AddMatchSignal(
		dbus.WithMatchSender("net.connman"),
		dbus.WithMatchInterface("net.connman.Service"),
		dbus.WithMatchMember("PropertyChanged"),
	); err != nil {
		logger.Fatalf("Failed to add D-Bus signal match: %v", err)
	}
	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)
	logger.Println("Listening for network state changes...")

	// Handle incoming signals
	for signal := range signals {
		// Expect PropertyChanged signature: (string name, variant value)
		if len(signal.Body) < 2 {
			continue
		}
		// Only handle State property changes
		prop, ok := signal.Body[0].(string)
		if !ok || prop != "State" {
			continue
		}
		// Extract the state value from variant
		variant, ok := signal.Body[1].(dbus.Variant)
		if !ok {
			logger.Printf("Unexpected variant type for State: %T", signal.Body[1])
			continue
		}
		stateVal := variant.Value()
		state, ok := stateVal.(string)
		if !ok {
			logger.Printf("Unexpected variant value type for State: %T", stateVal)
			continue
		}
		logger.Printf("Network state changed to: %s", state)

		// Fetch additional network details from the triggering service
		ssid, connType := getNetworkDetailsForService(conn, signal.Path)

		// Prepare environment variables
		envVars := map[string]string{
			"NETWORK_STATE":   state,
			"NETWORK_SSID":    ssid,
			"CONNECTION_TYPE": connType,
		}

		// Map ConnMan states to up/down actions: "ready" or "online" => up, "idle" or "offline" => down
		switch state {
		case "ready", "online":
			executeScripts(paths, "up", envVars)
		case "idle", "offline":
			executeScripts(paths, "down", envVars)
		}
	}
}

// stringSlice implements flag.Value to collect multiple -p flags.
type stringSlice []string

// String returns the string representation of the collected flags.
func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

// Set appends a new value to the stringSlice.
func (s *stringSlice) Set(val string) error {
	*s = append(*s, val)
	return nil
}

func main() {
	var paths stringSlice
	flag.Var(&paths, "p", "path to scripts directory; may be specified multiple times")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -p PATH [-p PATH]...\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No paths specified")
		flag.Usage()
		os.Exit(1)
	}

	listenForNetworkChanges([]string(paths))
}
