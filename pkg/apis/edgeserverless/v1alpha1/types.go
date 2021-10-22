package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Route struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              RouteSpec `json:"spec"`
}

type RouteSpec struct {
	ID      string        `json:"id,omitempty"`
	Name    string        `json:"name"`
	URI     string        `json:"uri"`
	Targets []RouteTarget `json:"targets"`
}

type RouteTarget struct {
	Target      string `json:"target"`
	Type        string `json:"type"`
	Ratio       int64  `json:"ratio"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RouteList struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Route `json:"items"`
}
