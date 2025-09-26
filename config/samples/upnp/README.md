# UPnP IGD Integration with MetalLB

This directory contains sample configurations for using UPnP IGD (Internet Gateway Device) port forwarding with MetalLB. UPnP IGD allows MetalLB to automatically configure port forwarding on compatible routers, enabling external access to LoadBalancer services without manual router configuration.

## Overview

UPnP IGD integration in MetalLB provides:

- Automatic port forwarding configuration on UPnP-enabled routers
- External IP address discovery and assignment
- Dynamic port mapping management
- Support for both TCP and UDP protocols
- Integration with MetalLB's existing pool and advertisement system

## Prerequisites

1. **Router with UPnP IGD support**: Your router must have UPnP IGD enabled in its configuration
2. **Network access**: The MetalLB speaker pod must be able to discover and communicate with the router
3. **Node selection**: At least one Kubernetes node must have network access to the router (typically the node on the same network segment)

## Configuration

### 1. UPnPAdvertisement Resource

The `UPnPAdvertisement` resource defines how MetalLB should handle UPnP port forwarding:

```yaml
apiVersion: metallb.io/v1beta1
kind: UPnPAdvertisement
metadata:
  name: upnp-advertisement
  namespace: metallb-system
spec:
  # Select IP address pools for UPnP forwarding
  ipAddressPools: ["upnp-pool"]
  
  # Or use label selectors
  ipAddressPoolSelectors:
  - matchLabels:
      metallb.universe.tf/upnp-enabled: "true"
  
  # Select nodes that can handle UPnP
  nodeSelectors:
  - matchLabels:
      metallb.universe.tf/upnp-node: "true"
  
  # Port mapping duration (seconds, 0 = permanent)
  duration: 0
  
  # Default description for port mappings
  description: "MetalLB LoadBalancer"
```

### 2. Node Labeling

Label the nodes that should handle UPnP forwarding:

```bash
kubectl label node <node-name> metallb.universe.tf/upnp-node=true
```

### 3. Service Configuration

Enable UPnP for specific services using annotations:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
  annotations:
    # Enable UPnP port forwarding
    metallb.universe.tf/upnp-enabled: "true"
    
    # Optional: Custom description
    metallb.universe.tf/upnp-description: "My Application"
    
    # Optional: Custom duration (seconds)
    metallb.universe.tf/upnp-duration: "86400"
spec:
  type: LoadBalancer
  ports:
  - port: 80
    protocol: TCP
  selector:
    app: my-app
```

## Service Annotations

| Annotation | Description | Default | Example |
|------------|-------------|---------|---------|
| `metallb.universe.tf/upnp-enabled` | Enable UPnP forwarding | `false` | `"true"` |
| `metallb.universe.tf/upnp-description` | Port mapping description | `"MetalLB LoadBalancer"` | `"Web Server"` |
| `metallb.universe.tf/upnp-duration` | Mapping duration in seconds | `0` (permanent) | `"3600"` |

## How It Works

1. **Discovery**: MetalLB discovers UPnP IGD devices on the local network
2. **Election**: One MetalLB speaker per pool is elected to handle UPnP forwarding
3. **External IP**: The external IP address is retrieved from the router
4. **Port Mapping**: Port mappings are created for each service port
5. **Status Reporting**: Service status includes external IP and port mappings

## Troubleshooting

### Check UPnP Device Discovery

Look for UPnP discovery messages in the MetalLB speaker logs:

```bash
kubectl logs -n metallb-system daemonset/speaker | grep -i upnp
```

### Verify Router Configuration

Ensure your router has:
- UPnP IGD enabled
- Port forwarding allowed
- No firewall blocking UPnP traffic

### Check Service Status

View the UPnP status for a service:

```bash
kubectl get serviceupnpstatus -n <namespace>
```

### Common Issues

1. **No UPnP device found**: Router doesn't support UPnP IGD or it's disabled
2. **Permission denied**: Router doesn't allow port forwarding via UPnP
3. **Port conflicts**: Requested external port is already in use
4. **Network isolation**: Speaker pod can't reach the router

## Security Considerations

- UPnP can be a security risk if not properly configured
- Only enable UPnP on trusted networks
- Consider using specific node selection to limit UPnP access
- Monitor external port mappings regularly
- Use non-permanent mappings when possible

## Example Deployment

See `upnpadvertisement_sample.yaml` for a complete example that includes:
- UPnPAdvertisement configuration
- IPAddressPool with UPnP labels
- Service with UPnP annotations
- Sample nginx deployment

## Limitations

- Only one MetalLB speaker per pool handles UPnP forwarding
- Router must support UPnP IGD protocol
- External IP is determined by the router
- Port conflicts are handled at the router level
- IPv6 support depends on router capabilities

## Integration with Other MetalLB Features

UPnP IGD works alongside:
- **Layer 2 (ARP)**: Can be used together for redundancy
- **BGP**: Complementary for different network segments  
- **Address pools**: Full integration with pool selection
- **Node selection**: Compatible with node-based filtering

For more information, see the MetalLB documentation at https://metallb.universe.tf/