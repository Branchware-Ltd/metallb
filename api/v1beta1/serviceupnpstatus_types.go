/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ServiceUPnPStatusSpec defines the observed state of the UPnP port forwarding for a service.
type ServiceUPnPStatusSpec struct {
	// ExternalIP is the external IP address provided by the UPnP IGD router
	ExternalIP string `json:"externalIP,omitempty"`
	// PortMappings contains the list of UPnP port mappings created for this service
	PortMappings []UPnPPortMapping `json:"portMappings,omitempty"`
	// Node is the name of the node that is handling the UPnP port forwarding
	Node string `json:"node,omitempty"`
}

// UPnPPortMapping represents a single UPnP port mapping
type UPnPPortMapping struct {
	// ExternalPort is the external port on the router
	ExternalPort int32 `json:"externalPort"`
	// InternalPort is the internal port on the service
	InternalPort int32 `json:"internalPort"`
	// Protocol is the protocol (TCP or UDP)
	Protocol string `json:"protocol"`
	// Description is the description of the port mapping
	Description string `json:"description,omitempty"`
	// Duration is the lease duration in seconds (0 means permanent)
	Duration int32 `json:"duration,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="External IP",type=string,JSONPath=`.spec.externalIP`
//+kubebuilder:printcolumn:name="Node",type=string,JSONPath=`.spec.node`
//+kubebuilder:printcolumn:name="Port Mappings",type=integer,JSONPath=`.spec.portMappings[*].externalPort`,priority=10

// ServiceUPnPStatus shows the current UPnP port forwarding status for a service.
type ServiceUPnPStatus struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ServiceUPnPStatusSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ServiceUPnPStatusList contains a list of ServiceUPnPStatus.
type ServiceUPnPStatusList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceUPnPStatus `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ServiceUPnPStatus{}, &ServiceUPnPStatusList{})
}
