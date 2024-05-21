package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	deploymentName = "router-default"
	namespace      = "openshift-ingress"
)

var lastEnvContent string

func main() {
	var envFilePath string

	// Parse the command-line arguments
	flag.StringVar(&envFilePath, "env-file", "/etc/profile.d/router-default.sh", "Path to the environment file")
	flag.Parse()

	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to local config
		if home := homedir.HomeDir(); home != "" {
			configPath := filepath.Join(home, ".kube", "config")
			config, err = clientcmd.BuildConfigFromFlags("", configPath)
			if err != nil {
				fmt.Printf("Error creating local config: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Error creating in-cluster config: %v\n", err)
			os.Exit(1)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	// Create a list watcher for the deployments
	deploymentListWatcher := cache.NewListWatchFromClient(
		clientset.AppsV1().RESTClient(),
		"deployments",
		namespace,
		fields.Everything(),
	)

	// Create an informer with options
	options := cache.SharedIndexInformerOptions{
		ResyncPeriod: 30 * time.Second,
	}
	informer := cache.NewSharedIndexInformer(
		deploymentListWatcher,
		&v1.Deployment{},
		options.ResyncPeriod,
		cache.Indexers{},
	)

	// Add event handlers to the informer
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment := obj.(*v1.Deployment)
			if deployment.Name == deploymentName {
				writeEnvFile(deployment, "AddFunc", envFilePath)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			deployment := newObj.(*v1.Deployment)
			if deployment.Name == deploymentName {
				writeEnvFile(deployment, "UpdateFunc", envFilePath)
			}
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	go informer.Run(stopCh)

	// Wait for signals to stop the program
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down...")
}

func writeEnvFile(deployment *v1.Deployment, event, envFilePath string) {
	envFileContent := extractEnvVars(deployment)

	if envFileContent == lastEnvContent {
		fmt.Printf("No changes in environment variables. Skipping file write. Event: %s\n", event)
		return
	}

	err := os.WriteFile(envFilePath, []byte(envFileContent), 0644)
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		os.Exit(1)
	}

	lastEnvContent = envFileContent
	fmt.Printf("Deployment %s/%s environment variables written to %s. Event: %s\n", namespace, deploymentName, envFilePath, event)
}

func extractEnvVars(deployment *v1.Deployment) string {
	var envFileContent string

	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			envFileContent += fmt.Sprintf("export %s=%s\n", env.Name, env.Value)
		}
	}

	return envFileContent
}
