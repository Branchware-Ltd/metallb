// SPDX-License-Identifier:Apache-2.0

package upnp

import (
	"net"
	"testing"

	"github.com/go-kit/log"
)

func TestPortMapping(t *testing.T) {
	// Test port mapping structure
	mapping := &PortMapping{
		ExternalPort: 8080,
		InternalPort: 80,
		InternalIP:   net.ParseIP("192.168.1.100"),
		Protocol:     "TCP",
		Description:  "Test Web Server",
		Duration:     3600,
	}

	if mapping.ExternalPort != 8080 {
		t.Errorf("Expected external port 8080, got %d", mapping.ExternalPort)
	}

	if mapping.InternalPort != 80 {
		t.Errorf("Expected internal port 80, got %d", mapping.InternalPort)
	}

	if !mapping.InternalIP.Equal(net.ParseIP("192.168.1.100")) {
		t.Errorf("Expected internal IP 192.168.1.100, got %s", mapping.InternalIP.String())
	}

	if mapping.Protocol != "TCP" {
		t.Errorf("Expected protocol TCP, got %s", mapping.Protocol)
	}

	if mapping.Description != "Test Web Server" {
		t.Errorf("Expected description 'Test Web Server', got %s", mapping.Description)
	}

	if mapping.Duration != 3600 {
		t.Errorf("Expected duration 3600, got %d", mapping.Duration)
	}
}

func TestNewIGDClient(t *testing.T) {
	logger := log.NewNopLogger()

	// This test will likely fail if no UPnP device is available
	// but we test the function signature and basic error handling
	_, err := New(logger)

	// We expect this to fail in most test environments
	// as there won't be a UPnP IGD device available
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestPortMappingValidation(t *testing.T) {
	logger := log.NewNopLogger()

	// Create a mock client structure for testing validation logic
	// In a real environment, this would have actual IGD clients
	client := &IGDClient{
		logger:  logger,
		version: 0, // No client available
	}

	// Test nil mapping
	err := client.AddPortMapping(nil)
	if err == nil {
		t.Error("Expected error for nil port mapping")
	}

	// Test invalid protocol
	invalidMapping := &PortMapping{
		ExternalPort: 8080,
		InternalPort: 80,
		InternalIP:   net.ParseIP("192.168.1.100"),
		Protocol:     "INVALID",
		Description:  "Test",
		Duration:     0,
	}

	err = client.AddPortMapping(invalidMapping)
	if err == nil {
		t.Error("Expected error for invalid protocol")
	}

	// Test valid TCP mapping
	tcpMapping := &PortMapping{
		ExternalPort: 8080,
		InternalPort: 80,
		InternalIP:   net.ParseIP("192.168.1.100"),
		Protocol:     "TCP",
		Description:  "Test TCP",
		Duration:     0,
	}

	err = client.AddPortMapping(tcpMapping)
	// This should fail because no IGD client is available
	if err == nil {
		t.Error("Expected error when no IGD client available")
	}

	// Test valid UDP mapping
	udpMapping := &PortMapping{
		ExternalPort: 8080,
		InternalPort: 80,
		InternalIP:   net.ParseIP("192.168.1.100"),
		Protocol:     "UDP",
		Description:  "Test UDP",
		Duration:     0,
	}

	err = client.AddPortMapping(udpMapping)
	// This should fail because no IGD client is available
	if err == nil {
		t.Error("Expected error when no IGD client available")
	}
}

func TestDeletePortMapping(t *testing.T) {
	logger := log.NewNopLogger()

	client := &IGDClient{
		logger:  logger,
		version: 0, // No client available
	}

	// Test invalid protocol
	err := client.DeletePortMapping(8080, "INVALID")
	if err == nil {
		t.Error("Expected error for invalid protocol")
	}

	// Test valid protocols but no client
	err = client.DeletePortMapping(8080, "TCP")
	if err == nil {
		t.Error("Expected error when no IGD client available")
	}

	err = client.DeletePortMapping(8080, "UDP")
	if err == nil {
		t.Error("Expected error when no IGD client available")
	}
}

func TestGetDeviceInfo(t *testing.T) {
	logger := log.NewNopLogger()

	client := &IGDClient{
		logger:  logger,
		version: 0, // No client available
	}

	info := client.GetDeviceInfo()
	if info == nil {
		t.Error("Expected non-nil device info map")
	}

	// Should be empty when no client is available
	if len(info) > 1 { // May contain external_ip key even when nil
		t.Errorf("Expected empty or minimal device info, got %v", info)
	}
}

func TestIsAvailable(t *testing.T) {
	logger := log.NewNopLogger()

	client := &IGDClient{
		logger:  logger,
		version: 0, // No client available
	}

	if client.IsAvailable() {
		t.Error("Expected client to not be available when no IGD clients are set")
	}

	// Test with mock version 1
	client.version = 1
	if client.IsAvailable() {
		t.Error("Expected client to not be available when client1 is nil")
	}

	// Test with mock version 2
	client.version = 2
	if client.IsAvailable() {
		t.Error("Expected client to not be available when client2 is nil")
	}
}

func TestGetExternalIP(t *testing.T) {
	logger := log.NewNopLogger()

	client := &IGDClient{
		logger: logger,
	}

	// Test when external IP is not set
	ip := client.GetExternalIP()
	if ip != nil {
		t.Errorf("Expected nil external IP, got %s", ip.String())
	}

	// Test with mock external IP
	client.externalIP = net.ParseIP("203.0.113.1")
	ip = client.GetExternalIP()
	if ip == nil {
		t.Error("Expected external IP to be set")
	}
	if !ip.Equal(net.ParseIP("203.0.113.1")) {
		t.Errorf("Expected external IP 203.0.113.1, got %s", ip.String())
	}
}
