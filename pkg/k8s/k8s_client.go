package pkg

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/astaxie/beego"
)

type RESTClient struct {
	DynamicRESTClients map[string]*DynamicRESTClient
}

type DynamicRESTClient struct {
	KubeRESTConfig    *rest.Config
	DynamicRESTClient dynamic.Interface
}

var (

	// get kube config path
	kubeConfigPath = beego.AppConfig.String("k8s::configpath")

	// get kube config with suffix
	kubeConfigSuffix = beego.AppConfig.String("k8s::configsuffix")

	// get kube datacenter
	datacenters = beego.AppConfig.Strings("k8s::datacenter")

	// RESTClienter
	RESTClienter *RESTClient

	dynamicRESTClienter *DynamicRESTClient
)

func NewRESTClient(dynDynamicRESTClients map[string]*DynamicRESTClient) *RESTClient {
	return &RESTClient{
		DynamicRESTClients: dynDynamicRESTClients,
	}
}

func NewDynamicRESTClient(KubeRESTConfig *rest.Config, dynDynamicRESTClient dynamic.Interface) *DynamicRESTClient {
	return &DynamicRESTClient{
		KubeRESTConfig:    KubeRESTConfig,
		DynamicRESTClient: dynDynamicRESTClient,
	}
}

func init() {

	//var drc dynamicRESTClienter
	var dynamicRESTClients = make(map[string]*DynamicRESTClient)
	if len(datacenters) == 0 {
		beego.BeeLogger.Warn("no datacenter was found")
	} else {
		for _, datacenter := range datacenters {
			kubeconfigLocation := kubeConfigPath + "/" + datacenter + kubeConfigSuffix
			kubeRESTConfig, dynamicClient, err := NewK8sClient(kubeconfigLocation)
			if err != nil {
				beego.BeeLogger.Warn("cloud not load the %s kube config", datacenter)
			} else {
				dynamicRESTClienter = NewDynamicRESTClient(kubeRESTConfig, dynamicClient)

				dynamicRESTClients[datacenter] = dynamicRESTClienter
				beego.BeeLogger.Info("Loading %s kube config succeed", datacenter)
			}
		}
	}
	RESTClienter = NewRESTClient(dynamicRESTClients)
}

// NewK8sClient  return k8s client according to datacenter
func NewK8sClient(kubeconfigLocation string) (*rest.Config, dynamic.Interface, error) {

	// The kubeconfig configuration file is loaded natively,
	// so the first parameter(masterURL) is an empty string
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigLocation)
	if err != nil {
		return nil, nil, err
	}

	// New dynamicREST client
	dynamicREST, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, err
	}
	return config, dynamicREST, nil
}
