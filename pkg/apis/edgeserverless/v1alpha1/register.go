package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"github.com/seveirbian/edgeserverless/pkg/apis/edgeserverless"
)

// GroupName is the group name use in this package
const GroupName = "edgeserverless.kubeedge.io"

// 注册自己的自定义资源
var SchemeGroupVersion = schema.GroupVersion{Group: edgeserverless.Group,
	Version: edgeserverless.Version}

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

var (
	// SchemeBuilder initializes a scheme builder
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	// AddToScheme is a global function that registers this API group & version to a scheme
	AddToScheme = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	// add Route and RouteList to scheme
	scheme.AddKnownTypes(SchemeGroupVersion,
		&Route{},
		&RouteList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
