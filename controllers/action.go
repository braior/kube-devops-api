package controllers

import (
	"context"
	"fmt"

	"github.com/astaxie/beego"

	pkg "github.com/braior/kube-devops-api/pkg/k8s"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

var (
	deploymentEntryType = EnterType{
		get:    "GetDeployment",
		list:   "ListDeployment",
		delete: "DeleteDeployment",
		create: "CreateDeployment",
	}
	decUnstructured = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
)

func (r *ResourceController) List() {
	resourceType := r.GetString("resourceType")
	datacenter := r.GetString("datacenter")
	namespace := r.GetString("namespace", "default")
	label := r.GetString("label", "")

	resourceList, err := list(r.Ctx.Request.Context(), resourceType, datacenter, namespace, label)

	if err != nil {
		r.Json(r.EntryType(resourceType), "failure", err.Error(), nil)
		return
	}

	r.Json(r.EntryType(resourceType), "success", "", resourceList)
}

func (r *ResourceController) Create() {
	resourceType := r.GetString("resourceType")
	datacenter := r.GetString("datacenter")
	namespace := r.GetString("namespace")

	requestData := r.Ctx.Input.RequestBody

	info, err := create(r.Ctx.Request.Context(), resourceType, datacenter, namespace, requestData)
	if err != nil {
		r.Json(r.EntryType(resourceType), "failure", err.Error(), nil)
		return
	}

	r.Json(r.EntryType(resourceType), "success", info, nil)
}

func (r *ResourceController) Get() {

	resourceType := r.GetString("resourceType")
	datacenter := r.GetString("datacenter")
	namespace := r.GetString("namespace", "default")
	name := r.GetString("name")

	resource, err := get(r.Ctx.Request.Context(), resourceType, datacenter, namespace, name)

	if err != nil {
		r.Json(r.EntryType(resourceType), "failure", err.Error(), nil)
		return
	}

	r.Json(r.EntryType(resourceType), "success", "", resource)
}

// func (r *ResourceController) Delete() {
// 	resourceType := r.GetString("resourceType")
// 	datacenter := r.GetString("datacenter")
// 	namespace := r.GetString("namespace", "default")
// 	name := r.GetString("name")

// }

func list(ctx context.Context, resourceType, datacenter, namespace, label string) (*appsv1.DeploymentList, error) {

	// check whether the datacenter is in the DynamicRESTClients
	rc, ok := pkg.RESTClienter.DynamicRESTClients[datacenter]
	if ok {
		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: resourceType}

		unstructObj, err := rc.DynamicRESTClient.
			Resource(gvr).
			Namespace(namespace).
			List(ctx, metav1.ListOptions{LabelSelector: label, Limit: 100})
		if err != nil {
			beego.Error(err)
			return nil, err
		}

		// Instantiate a deployment list data structure to receive the
		// results converted from unstructurobj
		deploymentList := &appsv1.DeploymentList{}

		// convert
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(
			unstructObj.UnstructuredContent(),
			deploymentList,
		)
		if err != nil {
			beego.Error(err)
			return nil, err
		}
		return deploymentList, nil
	}
	return nil, ErrNotFoundDatacenter(datacenter)
}

func get(ctx context.Context, resourceType, datacenter, namespace, name string) (*appsv1.Deployment, error) {

	// check whether the datacenter is in the DynamicRESTClients
	rc, ok := pkg.RESTClienter.DynamicRESTClients[datacenter]
	if ok {
		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: resourceType}
		unstructObj, err := rc.DynamicRESTClient.
			Resource(gvr).
			Namespace(namespace).
			Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			beego.Error(err)
			return nil, err
		}

		// Instantiate a deployment list data structure to receive the
		// results converted from unstructurobj
		deployment := &appsv1.Deployment{}

		// convert
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(
			unstructObj.UnstructuredContent(),
			deployment,
		)
		if err != nil {
			beego.Error(err)
			return nil, err
		}
		return deployment, nil
	}
	return nil, ErrNotFoundDatacenter(datacenter)
}

func create(ctx context.Context, resourceType, datacenter, namespace string, resource []byte) (string, error) {

	// check whether the datacenter is in the DynamicRESTClients
	rc, ok := pkg.RESTClienter.DynamicRESTClients[datacenter]
	if !ok {
		return "", ErrNotFoundDatacenter(datacenter)
	}

	// 1. Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(rc.KubeRESTConfig)
	if err != nil {
		beego.Error(err)
		return "", err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// 2. Decode YAML manifest into unstructured.Unstructured
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode([]byte(resource), nil, obj)
	if err != nil {
		return "", err
	}

	// 4. Find GVR
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", err
	}

	// 5. Obtain REST interface for the GVR
	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = rc.DynamicRESTClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		// for cluster-wide resources
		dr = rc.DynamicRESTClient.Resource(mapping.Resource)
	}

	// 6. Marshal object into JSON
	// data, err := json.Marshal(obj)
	// if err != nil {
	// 	return "", err
	// }

	// Create resource
	result, err := dr.Create(ctx, obj, metav1.CreateOptions{
		FieldManager: "sample-controller",
	})
	if err != nil {
		beego.Error(err)
		return "", err
	}

	beego.Info(fmt.Sprintf("created %s %q succeed", resourceType, result.GetName()))
	return result.GetName(), nil

}
