package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeRegisterSpec defines the desired state of NodeRegister
type NodeRegisterSpec struct {
	// Size is the size of the nodeRegister deployment
	Size       int32 `json:"size"`
	Port       int32 `json:"port"`
	TargetPort int32 `json:"target_port"`
}

// NodeRegisterStatus defines the observed state of NodeRegister
type NodeRegisterStatus struct {
	// Nodes are the names of the nodeRegister pods
	Nodes []string `json:"nodes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeRegister is the Schema for the noderegisters API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=noderegisters,scope=Namespaced
type NodeRegister struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeRegisterSpec   `json:"spec,omitempty"`
	Status NodeRegisterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeRegisterList contains a list of NodeRegister
type NodeRegisterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeRegister `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeRegister{}, &NodeRegisterList{})
}
