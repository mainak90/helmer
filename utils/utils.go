package utils

import (
	"database/sql"
	chartQueries "github.com/mainak90/helmer/queries/chart"
	"github.com/pkg/errors"
	"github.com/radovskyb/watcher"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
)


/// These are some helper functions to help the main controllers

// Helper function to check if a file exists..
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Get action config returns a kubernetes client to the the caller
func GetActionConfig(namespace string) (*action.Configuration, error) {
	var configFile string
	actionConfig := new(action.Configuration)
	// Incase this is run in-cluster, uses the serviceaccount to create resources, must have cluster
	// level rolebinding
	if FileExists("/var/run/secrets/kubernetes.io/serviceaccount/token") {
		log.Printf("In-cluster config detected!")
		var kubeConfig *genericclioptions.ConfigFlags
		// Create the rest config instance with ServiceAccount values loaded in them
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		// Create the ConfigFlags struct instance with initialized values from ServiceAccount
		kubeConfig = genericclioptions.NewConfigFlags(false)
		kubeConfig.APIServer = &config.Host
		kubeConfig.BearerToken = &config.BearerToken
		kubeConfig.CAFile = &config.CAFile
		kubeConfig.Namespace = &namespace
		if err := actionConfig.Init(kubeConfig, namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
			return nil, err
		}
		return actionConfig, nil
	} else if FileExists("/etc/rancher/k3s/k3s.yaml"){
		// Incase using k3s distro
		log.Printf("Rancher k3s config detected!")
		configFile = "/etc/rancher/k3s/k3s.yaml"
		if err := actionConfig.Init(kube.GetConfig(configFile, "", namespace), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
			return nil, err
		}
		return actionConfig, nil
	} else {
		// Create client from standard kubeconfig.
		log.Printf("Standard kubernetes config detected!")
		configFile = filepath.Join(os.Getenv("HOME"), ".kube", "config")
		if err := actionConfig.Init(kube.GetConfig(configFile, "", namespace), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
			return nil, err
		}
		return actionConfig, nil
	}
	return nil, nil
}

// Check if a helm release is installed already in the given namespace, incase method returns false,
// release deletion is not triggered to avoid unnecessary failures on logs
func IsInstalled(namespace string, releasename string) bool {

	actionConfig, err := GetActionConfig(namespace)

	if err != nil {
		panic(err)
	}

	listInstall := action.NewList(actionConfig)

	releases, err := listInstall.Run()

	if err != nil {
		log.Printf("Error encountered while getting release lists from cluster: %-v\n", err)
	}

	for _, release := range releases {
		log.Println("Release: " + release.Name + " Status: " + release.Info.Status.String())
		if release.Name == releasename {
			return true
		}
	}
	return false
}

// Check if chart metadata deems it installable.
func IsChartInstallable(ch *chart.Chart) (bool, error) {
	switch ch.Metadata.Type {
	case "", "application":
		return true, nil
	}
	return false, errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

// Standard watcher for deletion action of the chart directory, to avoid manual deletion of charts from
// fs while the database will continue to persist the tgz chartfile record unless deleted.
func WatchFile(db *sql.DB) {
	w := watcher.New()

	w.SetMaxEvents(1)

	w.FilterOps(watcher.Remove)

	go func() {
		for {
			select {
			case event := <-w.Event:
				log.Println(event)
				slices := strings.Split(event.Path, "/")
				// Resorted to use table row deletion on name and version as using path as field
				// doesn't work somehow.
				version := slices[len(slices) - 2]
				name := slices[len(slices) - 3]
				log.Printf("Removing record from database for chart in path %s", name)
				chartQuery := chartQueries.ChartQueries{}
				del := chartQuery.RemoveChart(db, name, version)
				log.Printf("Removed record from database for chart in path %s", name)
				log.Printf("Rows deleted: %s", del)
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	// Recursively checking if /tmp/charts has deletion somewhere, the path is hardcoded for now.
	if err := w.AddRecursive("/tmp/charts/"); err != nil {
		log.Fatalln(err)
	}

	for path, f := range w.WatchedFiles() {
		log.Printf("%s: %s\n", path, f.Name())
	}

	go func() {
		w.Wait()
	}()

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}

// No idea why this part is not working for now, focussing on the other topics
//func CreateNS(namespace string) error {
//	clientset := Client()
//	var ctx context.Context
//	ns := &corev1.Namespace{
//		ObjectMeta: metav1.ObjectMeta{
//			Name: namespace,
//			Labels: map[string]string{
//				"name": namespace,
//				},
//		},
//	}
//			_, err := clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
//			if err != nil {
//				log.Printf("Error encountered: %s", err)
//				return err
//		}
//		return nil
//}
//
//func Client() *kubernetes.Clientset {
//	var kubeconfig *string
//	if FileExists("/var/run/secrets/kubernetes.io/serviceaccount/token") {
//		log.Printf("Incluster config pickedup")
//		config, err := rest.InClusterConfig()
//		if err != nil {
//			log.Printf("Error encountered: %s", err)
//			panic(err.Error())
//		}
//
//		clientset, err := kubernetes.NewForConfig(config)
//		if err != nil {
//			log.Printf("Error encountered: %s", err)
//			panic(err.Error())
//		}
//		return clientset
//	} else {
//		log.Printf("Local kubeconfig picked up")
//		kubeconfig = flag.String("kubeconfig", filepath.Join(os.Getenv("HOME"), ".kube", "config"), "(optional) absolute path to the kubeconfig file")
//
//		flag.Parse()
//
//		config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
//
//		if err != nil {
//			fmt.Printf("The kubeconfig file cannot be loaded %v \n", err)
//			os.Exit(1)
//		}
//
//		clientset, err := kubernetes.NewForConfig(config)
//		if err != nil {
//			fmt.Printf("There is an error creating the client instance with the kubeconfig file %v \n", err)
//			os.Exit(1)
//		}
//		return clientset
//
//	}
//}
