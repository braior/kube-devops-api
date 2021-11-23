package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/astaxie/beego"

	pkg "github.com/braior/kube-devops-api/pkg/k8s"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	deploymentEntryType = EnterType{
		get:    "GetDeployment",
		list:   "ListDeployment",
		delete: "DeleteDeployment",
		create: "CreateDeployment",
	}
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
	namespace := r.GetString("namespace", "default")

	requestData := r.Ctx.Input.RequestBody

	err := create(r.Ctx.Request.Context(), resourceType, datacenter, namespace, requestData)
	if err != nil {
		r.Json(r.EntryType(resourceType), "failure", err.Error(), nil)
		return
	}

	r.Json(r.EntryType(resourceType), "success", fmt.Sprintf("%s create success", resourceType), nil)
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

func list(ctx context.Context, resourceType, datacenter, namespace, label string) (*appsv1.DeploymentList, error) {

	// check whether the datacenter is in the DynamicRESTClients
	dynamicREST, ok := pkg.DynamicRESTClients[datacenter]
	if ok {
		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: resourceType}

		unstructObj, err := dynamicREST.
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
	dynamicREST, ok := pkg.DynamicRESTClients[datacenter]
	if ok {
		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: resourceType}
		unstructObj, err := dynamicREST.
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

func create(ctx context.Context, resourceType, datacenter, namespace string, resource []byte) error {

	var unstructurobj map[string]interface{}
	err := json.Unmarshal(resource, &unstructurobj)
	if err != nil {
		beego.Error(err)
		return err
	}

	// check whether the datacenter is in the DynamicRESTClients
	dynamicREST, ok := pkg.DynamicRESTClients[datacenter]
	if ok {

		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: resourceType}
		deployment := &unstructured.Unstructured{
			Object: unstructurobj,
		}

		// Create resource
		result, err := dynamicREST.
			Resource(gvr).
			Namespace(namespace).
			Create(ctx, deployment, metav1.CreateOptions{})

		if err != nil {
			beego.Error(err)
			return err
		}

		beego.Info(fmt.Sprintf("created %s %q succeed", resourceType, result.GetName()))
		return nil
	}

	return ErrNotFoundDatacenter(datacenter)
}
