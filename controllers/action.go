package controllers

import (
	"context"
	"fmt"
	"github.com/astaxie/beego/logs"

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
	resourceKind := r.GetString("resourceKind")
	datacenter := r.GetString("datacenter")
	namespace := r.GetString("namespace", "default")
	label := r.GetString("label", "")

	resourceList, err := list(r.Ctx.Request.Context(), resourceKind, datacenter, namespace, label)

	if err != nil {
		r.Json(r.EntryType(resourceKind), "failure", err.Error(), nil)
		return
	}

	r.Json(r.EntryType(resourceKind), "success", "", resourceList)
}

func (r *ResourceController) Create() {
	resourceKind := r.GetString("resourceKind")
	datacenter := r.GetString("datacenter")
	namespace := r.GetString("namespace")

	requestData := r.Ctx.Input.RequestBody

	info, err := create(r.Ctx.Request.Context(), resourceKind, datacenter, namespace, requestData)
	if err != nil {
		r.Json(r.EntryType(resourceKind), "failure", err.Error(), nil)
		return
	}

	r.Json(r.EntryType(resourceKind), "success", info, nil)
}

func (r *ResourceController) Get() {

	resourceKind := r.GetString("resourceKind")
	datacenter := r.GetString("datacenter")
	namespace := r.GetString("namespace", "default")
	name := r.GetString("name")

	resource, err := get(r.Ctx.Request.Context(), resourceKind, datacenter, namespace, name)

	if err != nil {
		r.Json(r.EntryType(resourceKind), "failure", err.Error(), nil)
		return
	}

	r.Json(r.EntryType(resourceKind), "success", "", resource)
}

// list is show the resource deploy detail
func list(ctx context.Context, resourceKind, datacenter, namespace, label string) (*appsv1.DeploymentList, error) {

	drc, gvk, gvr, err := getGVR(resourceKind, datacenter)
	if err != nil {
		return nil, err
	}

	unStructObj, err := drc.DynamicRESTClient.
		Resource(*gvr).
		Namespace(namespace).
		List(ctx, metav1.ListOptions{LabelSelector: label, Limit: 100})
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	// Instantiate a deployment list data structure to receive the
	// results converted from unStructurobj
	switch gvk.Kind {
	case "deployment":
		deploymentList := &appsv1.DeploymentList{}

		// convert
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(
			unStructObj.UnstructuredContent(),
			deploymentList,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}
		return deploymentList, nil
	case "pod":
	default:
	}
	return nil, err
}

func get(ctx context.Context, resourceKind, datacenter, namespace, name string) (*appsv1.Deployment, error) {

	drc, gvk, gvr, err := getGVR(resourceKind, datacenter)
	if err != nil {
		return nil, err
	}

	unstructObj, err := drc.DynamicRESTClient.
		Resource(*gvr).
		Namespace(namespace).
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	// Instantiate a deployment list data structure to receive the
	// results converted from unstructurobj
	switch gvk.Kind {
	case "deployment":
		deployment := &appsv1.Deployment{}

		// convert
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(
			unstructObj.UnstructuredContent(),
			deployment,
		)
		if err != nil {
			logs.Error(err)
			return nil, err
		}
		return deployment, nil
	case "pod":
	default:
	}
	return nil, err
}

func create(ctx context.Context, resourceKind, datacenter, namespace string, resource []byte) (string, error) {

	// check whether the datacenter is in the DynamicRESTClients
	drc, ok := pkg.RESTClienter.DynamicRESTClients[datacenter]
	if !ok {
		return "", ErrNotFoundDatacenter(datacenter)
	}

	// 1. Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(drc.KubeRESTConfig)
	if err != nil {
		logs.Error(err)
		return "", err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// 2. Decode YAML manifest into unstructured.Unstructured
	obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(resource, nil, obj)
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
		dr = drc.DynamicRESTClient.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		// for cluster-wide resources
		dr = drc.DynamicRESTClient.Resource(mapping.Resource)
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
		logs.Error(err)
		return "", err
	}

	logs.Info(fmt.Sprintf("created %s %q succeed", resourceKind, result.GetName()))
	return result.GetName(), nil

}

func getGVR(resourceKind, datacenter string) (*pkg.DynamicRESTClient, *schema.GroupVersionKind, *schema.GroupVersionResource, error) {

	var rk *schema.GroupVersionKind
	rk.Kind = resourceKind
	// check whether the datacenter is in the DynamicRESTClients
	drc, ok := pkg.RESTClienter.DynamicRESTClients[datacenter]
	if !ok {
		return nil, nil, nil, ErrNotFoundDatacenter(datacenter)
	}

	// 1. Prepare a RESTMapper to find GVR
	dc, err := discovery.NewDiscoveryClientForConfig(drc.KubeRESTConfig)
	if err != nil {
		logs.Error(err)
		return drc, nil, nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(dc))

	// 2. Decode YAML manifest into unstructured.Unstructured
	//obj := &unstructured.Unstructured{}
	_, gvk, err := decUnstructured.Decode(nil, rk, nil)
	if err != nil {
		return drc, gvk, nil, err
	}

	// 4. Find GVR
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return drc, gvk, nil, err
	}

	return drc, gvk, &mapping.Resource, nil
}
