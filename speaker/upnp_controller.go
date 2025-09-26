// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	v1 "k8s.io/api/core/v1"
	discovery "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"go.universe.tf/metallb/internal/config"
	"go.universe.tf/metallb/internal/upnp"
)

const (
	// Annotation to enable UPnP IGD port forwarding
	UPnPEnabledAnnotation = "metallb.universe.tf/upnp-enabled"
	// Annotation to set UPnP port mapping description
	UPnPDescriptionAnnotation = "metallb.universe.tf/upnp-description"
	// Annotation to set UPnP port mapping duration
	UPnPDurationAnnotation = "metallb.universe.tf/upnp-duration"
	// Default UPnP port mapping description
	DefaultUPnPDescription = "MetalLB LoadBalancer"
	// Default UPnP port mapping duration (0 = permanent)
	DefaultUPnPDuration = 0
)

type upnpController struct {
	myNode         string
	client         *upnp.IGDClient
	mappings       map[string][]*upnp.PortMapping // service key -> port mappings
	config         *config.Config
	mutex          sync.RWMutex
	onStatusChange func(types.NamespacedName)
}

func newUPnPController(logger log.Logger, myNode string, onStatusChange func(types.NamespacedName)) (*upnpController, error) {
	// Try to create UPnP IGD client
	client, err := upnp.New(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize UPnP IGD client: %v", err)
	}

	controller := &upnpController{
		myNode:         myNode,
		client:         client,
		mappings:       make(map[string][]*upnp.PortMapping),
		onStatusChange: onStatusChange,
	}

	level.Info(logger).Log("msg", "UPnP IGD controller initialized", "external_ip", client.GetExternalIP())
	return controller, nil
}

func (c *upnpController) SetConfig(l log.Logger, cfg *config.Config) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.config = cfg
	return nil
}

func (c *upnpController) ShouldAnnounce(l log.Logger, name string, toAnnounce []net.IP, pool *config.Pool, svc *v1.Service, eps []discovery.EndpointSlice, nodes map[string]*v1.Node) string {
	// Check if UPnP is enabled for this service
	if !c.isUPnPEnabled(svc) {
		level.Debug(l).Log("event", "shouldannounce", "protocol", "upnp", "message", "UPnP not enabled for service", "service", name)
		return "notOwner"
	}

	// Check if UPnP is configured for this pool
	if !c.poolSupportsUPnP(pool) {
		level.Debug(l).Log("event", "shouldannounce", "protocol", "upnp", "message", "pool does not support UPnP", "service", name, "pool", pool.Name)
		return "notOwner"
	}

	// Check if we have active endpoints
	if !activeEndpointExists(eps) {
		level.Debug(l).Log("event", "shouldannounce", "protocol", "upnp", "message", "no active endpoints", "service", name)
		return "notOwner"
	}

	// Check if this node matches the pool's UPnP advertisement configuration
	if !c.nodeMatchesUPnPAdvertisements(pool) {
		level.Debug(l).Log("event", "shouldannounce", "protocol", "upnp", "message", "node does not match UPnP advertisements", "service", name, "node", c.myNode)
		return "notOwner"
	}

	// UPnP is typically handled by a single node (the one with access to the router)
	// We'll use a simple election based on node name lexicographic ordering
	eligibleNodes := c.getEligibleNodes(pool, nodes)
	if len(eligibleNodes) == 0 {
		level.Debug(l).Log("event", "shouldannounce", "protocol", "upnp", "message", "no eligible nodes", "service", name)
		return "notOwner"
	}

	// Sort nodes and check if we're the first one (simple leader election)
	if eligibleNodes[0] != c.myNode {
		level.Debug(l).Log("event", "shouldannounce", "protocol", "upnp", "message", "not the elected node for UPnP", "service", name, "elected", eligibleNodes[0], "mynode", c.myNode)
		return "notOwner"
	}

	level.Debug(l).Log("event", "shouldannounce", "protocol", "upnp", "message", "elected for UPnP forwarding", "service", name)
	return ""
}

func (c *upnpController) SetBalancer(l log.Logger, name string, lbIPs []net.IP, pool *config.Pool, client service, svc *v1.Service) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.isUPnPEnabled(svc) {
		level.Debug(l).Log("msg", "UPnP not enabled for service", "service", name)
		return nil
	}

	// Clean up existing mappings for this service
	if err := c.deleteExistingMappings(l, name); err != nil {
		level.Warn(l).Log("msg", "failed to clean up existing UPnP mappings", "service", name, "error", err)
	}

	// Create new port mappings
	var newMappings []*upnp.PortMapping

	for _, lbIP := range lbIPs {
		for _, port := range svc.Spec.Ports {
			// Skip headless services
			if port.Port == 0 {
				continue
			}

			protocol := strings.ToLower(string(port.Protocol))
			if protocol != "tcp" && protocol != "udp" {
				level.Warn(l).Log("msg", "unsupported protocol for UPnP", "protocol", protocol, "service", name)
				continue
			}

			mapping := &upnp.PortMapping{
				ExternalPort: int(port.Port),
				InternalPort: int(port.Port),
				InternalIP:   lbIP,
				Protocol:     protocol,
				Description:  c.getPortMappingDescription(svc, port),
				Duration:     c.getPortMappingDuration(svc),
			}

			if err := c.client.AddPortMapping(mapping); err != nil {
				level.Error(l).Log("msg", "failed to add UPnP port mapping", "service", name, "external_port", mapping.ExternalPort, "internal_ip", mapping.InternalIP, "error", err)
				client.Errorf(svc, "UPnPMappingFailed", "Failed to create UPnP port mapping for port %d: %v", mapping.ExternalPort, err)
				continue
			}

			newMappings = append(newMappings, mapping)
			level.Info(l).Log("msg", "created UPnP port mapping", "service", name, "external_port", mapping.ExternalPort, "internal_ip", mapping.InternalIP, "protocol", mapping.Protocol)
		}
	}

	// Store the new mappings
	c.mappings[name] = newMappings

	if len(newMappings) > 0 {
		client.Infof(svc, "UPnPMappingCreated", "Created %d UPnP port mapping(s) with external IP %s", len(newMappings), c.client.GetExternalIP())
		c.onStatusChange(types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace})
	}

	return nil
}

func (c *upnpController) DeleteBalancer(l log.Logger, name, reason string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if err := c.deleteExistingMappings(l, name); err != nil {
		level.Error(l).Log("msg", "failed to delete UPnP mappings", "service", name, "reason", reason, "error", err)
		return err
	}

	svcNamespace, svcName, err := cache.SplitMetaNamespaceKey(name)
	if err != nil {
		level.Warn(l).Log("op", "DeleteBalancer", "protocol", "upnp", "service", name, "msg", "failed to split key", "err", err)
		return err
	}
	c.onStatusChange(types.NamespacedName{Name: svcName, Namespace: svcNamespace})

	level.Info(l).Log("msg", "deleted UPnP port mappings", "service", name, "reason", reason)
	return nil
}

func (c *upnpController) SetNode(l log.Logger, n *v1.Node) error {
	// UPnP controller doesn't need to react to node changes specifically
	return nil
}

func (c *upnpController) SetEventCallback(callback func(interface{})) {
	// UPnP controller doesn't use event callbacks currently
}

// Helper methods

func (c *upnpController) isUPnPEnabled(svc *v1.Service) bool {
	if svc.Annotations == nil {
		return false
	}

	enabled, exists := svc.Annotations[UPnPEnabledAnnotation]
	if !exists {
		return false
	}

	return strings.ToLower(enabled) == "true"
}

func (c *upnpController) poolSupportsUPnP(pool *config.Pool) bool {
	return len(pool.UPnPAdvertisements) > 0
}

func (c *upnpController) nodeMatchesUPnPAdvertisements(pool *config.Pool) bool {
	for _, adv := range pool.UPnPAdvertisements {
		if !adv.Enabled {
			continue
		}
		if adv.Nodes[c.myNode] {
			return true
		}
	}
	return false
}

func (c *upnpController) getEligibleNodes(pool *config.Pool, nodes map[string]*v1.Node) []string {
	var eligible []string

	for _, adv := range pool.UPnPAdvertisements {
		if !adv.Enabled {
			continue
		}
		for nodeName := range adv.Nodes {
			if _, exists := nodes[nodeName]; exists {
				// Check if node is ready and not excluded from load balancers
				node := nodes[nodeName]
				if isNodeReady(node) && !isNodeExcludedFromBalancers(node) {
					eligible = append(eligible, nodeName)
				}
			}
		}
	}

	// Sort for consistent ordering
	for i := 0; i < len(eligible)-1; i++ {
		for j := i + 1; j < len(eligible); j++ {
			if eligible[i] > eligible[j] {
				eligible[i], eligible[j] = eligible[j], eligible[i]
			}
		}
	}

	return eligible
}

func (c *upnpController) deleteExistingMappings(l log.Logger, name string) error {
	mappings, exists := c.mappings[name]
	if !exists {
		return nil
	}

	for _, mapping := range mappings {
		if err := c.client.DeletePortMapping(mapping.ExternalPort, mapping.Protocol); err != nil {
			level.Warn(l).Log("msg", "failed to delete UPnP port mapping", "service", name, "external_port", mapping.ExternalPort, "protocol", mapping.Protocol, "error", err)
		}
	}

	delete(c.mappings, name)
	return nil
}

func (c *upnpController) getPortMappingDescription(svc *v1.Service, port v1.ServicePort) string {
	if svc.Annotations != nil {
		if desc, exists := svc.Annotations[UPnPDescriptionAnnotation]; exists && desc != "" {
			return fmt.Sprintf("%s (%s:%d)", desc, svc.Name, port.Port)
		}
	}
	return fmt.Sprintf("%s (%s:%d)", DefaultUPnPDescription, svc.Name, port.Port)
}

func (c *upnpController) getPortMappingDuration(svc *v1.Service) int {
	if svc.Annotations != nil {
		if durationStr, exists := svc.Annotations[UPnPDurationAnnotation]; exists {
			if duration, err := strconv.Atoi(durationStr); err == nil && duration >= 0 {
				return duration
			}
		}
	}
	return DefaultUPnPDuration
}

// Utility functions for node status checking

func isNodeReady(node *v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == v1.NodeReady {
			return condition.Status == v1.ConditionTrue
		}
	}
	return false
}

func isNodeExcludedFromBalancers(node *v1.Node) bool {
	if node.Labels != nil {
		if _, exists := node.Labels["node.kubernetes.io/exclude-from-external-load-balancers"]; exists {
			return true
		}
	}
	return false
}
