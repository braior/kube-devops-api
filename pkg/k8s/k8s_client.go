package pkg

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/astaxie/beego"
)

var (

	// get kube config path
	kubeConfigPath = beego.AppConfig.String("k8s::configpath")

	// get kube config with suffix
	kubeConfigFileNameSuffix = beego.AppConfig.String("k8s::configsuffix")

	// get kube datacenter
	datacenters = beego.AppConfig.Strings("k8s::datacenter")

	// DemonDynamicClient set the varaiable of datacenter demon
	DynamicClients = make(map[string]dynamic.Interface)
)

func init() {

	if len(datacenters) == 0 {
		beego.BeeLogger.Warn("no datacenter was found")
	} else {
		for _, datacenter := range datacenters {
			kubeconfigLocation := kubeConfigPath + datacenter + kubeConfigFileNameSuffix
			dynamicClient, err := NewK8sClient(kubeconfigLocation)
			if err != nil {
				beego.BeeLogger.Warn("cloud not get the %s config", datacenter)
			} else {
				DynamicClients[datacenter] = dynamicClient
				beego.BeeLogger.Info("Loading %s datacenter config succeed")
			}

		}

	}
}

// NewK8sClient  return k8s client according to datacenter
func NewK8sClient(kubeconfigLocation string) (dynamic.Interface, error) {

	// The kubeconfig configuration file is loaded natively,
	// so the first parameter(masterURL) is an empty string
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigLocation)
	if err != nil {
		return nil, err
	}

	// New dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return dynamicClient, nil
}
