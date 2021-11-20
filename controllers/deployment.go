package controllers

import (
	"context"
	"errors"

	"github.com/astaxie/beego"
	"github.com/braior/kube-devops-api/pkg"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (d *DeploymentController) Get() {
	datacenter := d.GetString("region")
	namespace := d.GetString("namespace", "default")
	lable := d.GetString("lable")

	deploymentList, err := getDeployment(d.Ctx.Request.Context(), datacenter, namespace, lable)
	if err != nil {
		beego.BeeLogger.Error(err.Error())
	}

	d.Data["json"] = *deploymentList
	d.ServeJSON()
}

func getDeployment(ctx context.Context, datacenter, namespace, lable string) (*appsv1.DeploymentList, error) {

	dynamicClient, ok := pkg.DynamicClients[datacenter]
	if ok {

		gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

		unstructObj, err := dynamicClient.
			Resource(gvr).
			Namespace(namespace).
			List(ctx, metav1.ListOptions{LabelSelector: lable, Limit: 100})
		if err != nil {
			beego.BeeLogger.Error(err.Error())
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
			panic(err.Error())
		}
		return deploymentList, nil
	}
	err := errors.New("There is no match to the existing datacenter")
	return nil, err
}
