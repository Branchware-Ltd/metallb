// SPDX-License-Identifier:Apache-2.0

package upnp

import (
	"fmt"
	"net"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/huin/goupnp/dcps/internetgateway1"
	"github.com/huin/goupnp/dcps/internetgateway2"
)

// IGDClient provides UPnP IGD (Internet Gateway Device) functionality
type IGDClient struct {
	logger     log.Logger
	client1    *internetgateway1.WANIPConnection1
	client2    *internetgateway2.WANIPConnection1
	version    int
	externalIP net.IP
}

// PortMapping represents a UPnP port mapping
type PortMapping struct {
	ExternalPort int
	InternalPort int
	InternalIP   net.IP
	Protocol     string
	Description  string
	Duration     int // Duration in seconds, 0 means permanent
}

// New creates a new UPnP IGD client
func New(logger log.Logger) (*IGDClient, error) {
	client := &IGDClient{
		logger: logger,
	}

	// Try IGD2 first (more modern)
	if err := client.discoverIGD2(); err != nil {
		level.Debug(logger).Log("msg", "IGD2 discovery failed, trying IGD1", "error", err)

		// Fallback to IGD1
		if err := client.discoverIGD1(); err != nil {
			return nil, fmt.Errorf("failed to discover UPnP IGD device: %v", err)
		}
	}

	// Get external IP address
	if err := client.updateExternalIP(); err != nil {
		level.Warn(logger).Log("msg", "failed to get external IP address", "error", err)
	}

	return client, nil
}

// discoverIGD2 attempts to discover IGD2 devices
func (c *IGDClient) discoverIGD2() error {
	clients, _, err := internetgateway2.NewWANIPConnection1Clients()
	if err != nil {
		return fmt.Errorf("failed to discover IGD2 clients: %v", err)
	}

	if len(clients) == 0 {
		return fmt.Errorf("no IGD2 devices found")
	}

	c.client2 = clients[0]
	c.version = 2
	level.Info(c.logger).Log("msg", "discovered UPnP IGD2 device", "location", c.client2.Location)
	return nil
}

// discoverIGD1 attempts to discover IGD1 devices
func (c *IGDClient) discoverIGD1() error {
	clients, _, err := internetgateway1.NewWANIPConnection1Clients()
	if err != nil {
		return fmt.Errorf("failed to discover IGD1 clients: %v", err)
	}

	if len(clients) == 0 {
		return fmt.Errorf("no IGD1 devices found")
	}

	c.client1 = clients[0]
	c.version = 1
	level.Info(c.logger).Log("msg", "discovered UPnP IGD1 device", "location", c.client1.Location)
	return nil
}

// updateExternalIP retrieves and caches the external IP address
func (c *IGDClient) updateExternalIP() error {
	var externalIP string
	var err error

	if c.version == 2 && c.client2 != nil {
		externalIP, err = c.client2.GetExternalIPAddress()
	} else if c.version == 1 && c.client1 != nil {
		externalIP, err = c.client1.GetExternalIPAddress()
	} else {
		return fmt.Errorf("no IGD client available")
	}

	if err != nil {
		return fmt.Errorf("failed to get external IP: %v", err)
	}

	c.externalIP = net.ParseIP(externalIP)
	if c.externalIP == nil {
		return fmt.Errorf("invalid external IP address: %s", externalIP)
	}

	level.Info(c.logger).Log("msg", "retrieved external IP address", "ip", c.externalIP.String())
	return nil
}

// GetExternalIP returns the external IP address
func (c *IGDClient) GetExternalIP() net.IP {
	return c.externalIP
}

// AddPortMapping creates a new port mapping
func (c *IGDClient) AddPortMapping(mapping *PortMapping) error {
	if mapping == nil {
		return fmt.Errorf("port mapping cannot be nil")
	}

	protocol := strings.ToUpper(mapping.Protocol)
	if protocol != "TCP" && protocol != "UDP" {
		return fmt.Errorf("invalid protocol: %s (must be TCP or UDP)", mapping.Protocol)
	}

	level.Debug(c.logger).Log(
		"msg", "adding port mapping",
		"external_port", mapping.ExternalPort,
		"internal_ip", mapping.InternalIP.String(),
		"internal_port", mapping.InternalPort,
		"protocol", protocol,
		"description", mapping.Description,
		"duration", mapping.Duration,
	)

	var err error
	if c.version == 2 && c.client2 != nil {
		err = c.client2.AddPortMapping(
			"", // RemoteHost (empty for any)
			uint16(mapping.ExternalPort),
			protocol,
			uint16(mapping.InternalPort),
			mapping.InternalIP.String(),
			true, // Enabled
			mapping.Description,
			uint32(mapping.Duration),
		)
	} else if c.version == 1 && c.client1 != nil {
		err = c.client1.AddPortMapping(
			"", // RemoteHost (empty for any)
			uint16(mapping.ExternalPort),
			protocol,
			uint16(mapping.InternalPort),
			mapping.InternalIP.String(),
			true, // Enabled
			mapping.Description,
			uint32(mapping.Duration),
		)
	} else {
		return fmt.Errorf("no IGD client available")
	}

	if err != nil {
		return fmt.Errorf("failed to add port mapping: %v", err)
	}

	level.Info(c.logger).Log(
		"msg", "successfully added port mapping",
		"external_port", mapping.ExternalPort,
		"internal_ip", mapping.InternalIP.String(),
		"internal_port", mapping.InternalPort,
		"protocol", protocol,
	)

	return nil
}

// DeletePortMapping removes a port mapping
func (c *IGDClient) DeletePortMapping(externalPort int, protocol string) error {
	protocol = strings.ToUpper(protocol)
	if protocol != "TCP" && protocol != "UDP" {
		return fmt.Errorf("invalid protocol: %s (must be TCP or UDP)", protocol)
	}

	level.Debug(c.logger).Log(
		"msg", "deleting port mapping",
		"external_port", externalPort,
		"protocol", protocol,
	)

	var err error
	if c.version == 2 && c.client2 != nil {
		err = c.client2.DeletePortMapping(
			"", // RemoteHost (empty for any)
			uint16(externalPort),
			protocol,
		)
	} else if c.version == 1 && c.client1 != nil {
		err = c.client1.DeletePortMapping(
			"", // RemoteHost (empty for any)
			uint16(externalPort),
			protocol,
		)
	} else {
		return fmt.Errorf("no IGD client available")
	}

	if err != nil {
		// Check if the error is because the mapping doesn't exist
		if strings.Contains(err.Error(), "NoSuchEntryInArray") ||
			strings.Contains(err.Error(), "SpecifiedArrayIndexInvalid") {
			level.Debug(c.logger).Log(
				"msg", "port mapping does not exist",
				"external_port", externalPort,
				"protocol", protocol,
			)
			return nil
		}
		return fmt.Errorf("failed to delete port mapping: %v", err)
	}

	level.Info(c.logger).Log(
		"msg", "successfully deleted port mapping",
		"external_port", externalPort,
		"protocol", protocol,
	)

	return nil
}

// GetPortMappings retrieves all current port mappings
func (c *IGDClient) GetPortMappings() ([]*PortMapping, error) {
	var mappings []*PortMapping
	index := 0

	for {
		var (
			externalPort  uint16
			protocol      string
			internalPort  uint16
			internalIP    string
			enabled       bool
			description   string
			leaseDuration uint32
			err           error
		)

		if c.version == 2 && c.client2 != nil {
			_, externalPort, protocol, internalPort, internalIP, enabled, description, leaseDuration, err =
				c.client2.GetGenericPortMappingEntry(uint16(index))
		} else if c.version == 1 && c.client1 != nil {
			_, externalPort, protocol, internalPort, internalIP, enabled, description, leaseDuration, err =
				c.client1.GetGenericPortMappingEntry(uint16(index))
		} else {
			return nil, fmt.Errorf("no IGD client available")
		}

		if err != nil {
			// End of list reached
			if strings.Contains(err.Error(), "SpecifiedArrayIndexInvalid") ||
				strings.Contains(err.Error(), "NoSuchEntryInArray") {
				break
			}
			return nil, fmt.Errorf("failed to get port mapping entry %d: %v", index, err)
		}

		if enabled {
			ip := net.ParseIP(internalIP)
			if ip != nil {
				mapping := &PortMapping{
					ExternalPort: int(externalPort),
					InternalPort: int(internalPort),
					InternalIP:   ip,
					Protocol:     protocol,
					Description:  description,
					Duration:     int(leaseDuration),
				}
				mappings = append(mappings, mapping)
			}
		}

		index++
		if index > 1000 { // Safety limit
			break
		}
	}

	level.Debug(c.logger).Log("msg", "retrieved port mappings", "count", len(mappings))
	return mappings, nil
}

// RefreshExternalIP updates the cached external IP address
func (c *IGDClient) RefreshExternalIP() error {
	return c.updateExternalIP()
}

// IsAvailable checks if the UPnP IGD client is available and functional
func (c *IGDClient) IsAvailable() bool {
	return (c.version == 1 && c.client1 != nil) || (c.version == 2 && c.client2 != nil)
}

// GetDeviceInfo returns information about the IGD device
func (c *IGDClient) GetDeviceInfo() map[string]interface{} {
	info := make(map[string]interface{})

	if c.version == 2 && c.client2 != nil {
		info["version"] = 2
		info["location"] = c.client2.Location
		info["device_url"] = c.client2.ServiceClient.RootDevice.URLBase.String()
	} else if c.version == 1 && c.client1 != nil {
		info["version"] = 1
		info["location"] = c.client1.Location
		info["device_url"] = c.client1.ServiceClient.RootDevice.URLBase.String()
	}

	if c.externalIP != nil {
		info["external_ip"] = c.externalIP.String()
	}

	return info
}
