package utils

import (
	"flag"
	"log"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func NewK8sClient() (dynamic.Interface, error) {
	var kubeconfig *string

	// if you can get the value of home directory, it can be used as the default value
	if home := homedir.HomeDir(); home != "" {
		// If the kubeconfig parameter is entered,
		// the value of this parameter is the absolute path of the kubeconfig file
		// If the kubeconfig parameter is not entered,
		// the default path ~/.Kube/config is used
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	log.SetFlags(log.Llongfile)
	flag.Parse()

	// The kubeconfig configuration file is loaded natively,
	// so the first parameter(masterURL) is an empty string
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	// New dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return dynamicClient, nil
}
