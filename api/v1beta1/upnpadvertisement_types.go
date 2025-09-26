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
// NOTE: json tags are required. Any new fields you add must have json tags for the fields to be serialized.

// UPnPAdvertisementSpec defines the desired state of UPnPAdvertisement.
type UPnPAdvertisementSpec struct {
	// The list of IPAddressPools to advertise via this advertisement, selected by name.
	// +optional
	IPAddressPools []string `json:"ipAddressPools,omitempty"`
	// A selector for the IPAddressPools which would get advertised via this advertisement.
	// If no IPAddressPool is selected by this or by the list, the advertisement is applied to all the IPAddressPools.
	// +optional
	IPAddressPoolSelectors []metav1.LabelSelector `json:"ipAddressPoolSelectors,omitempty"`
	// NodeSelectors allows to limit the nodes to announce as next hops for the LoadBalancer IP. When empty, all the nodes having  are announced as next hops.
	// +optional
	NodeSelectors []metav1.LabelSelector `json:"nodeSelectors,omitempty"`
	// Duration specifies the lease duration for UPnP port mappings in seconds.
	// 0 means permanent mappings (recommended for most use cases).
	// +optional
	// +kubebuilder:default=0
	Duration int `json:"duration,omitempty"`
	// Description is the default description for UPnP port mappings.
	// Individual service annotations can override this.
	// +optional
	// +kubebuilder:default="MetalLB LoadBalancer"
	Description string `json:"description,omitempty"`
}

// UPnPAdvertisementStatus defines the observed state of UPnPAdvertisement.
type UPnPAdvertisementStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="IPAddressPools",type=string,JSONPath=`.spec.ipAddressPools`
//+kubebuilder:printcolumn:name="IPAddressPool Selectors",type=string,JSONPath=`.spec.ipAddressPoolSelectors`
//+kubebuilder:printcolumn:name="Node Selectors",type=string,JSONPath=`.spec.nodeSelectors`,priority=10
//+kubebuilder:printcolumn:name="Duration",type=integer,JSONPath=`.spec.duration`

// UPnPAdvertisement allows to advertise the LoadBalancer IPs provided
// by the selected pools via UPnP IGD port forwarding.
type UPnPAdvertisement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UPnPAdvertisementSpec   `json:"spec,omitempty"`
	Status UPnPAdvertisementStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UPnPAdvertisementList contains a list of UPnPAdvertisement.
type UPnPAdvertisementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UPnPAdvertisement `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UPnPAdvertisement{}, &UPnPAdvertisementList{})
}
