package main

import (
	"fmt"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	RbacV1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func getNamespaceChannel(k *kubernetes.Clientset) <-chan watch.Event {
	namespaceWatch, err := k.CoreV1().Namespaces().Watch(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}

	return namespaceWatch.ResultChan()
}

func main() {

	kubeconfigPath, ok := os.LookupEnv("KUBECONFIG")

	var config *rest.Config
	var err error

	if !ok {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}

	if err != nil {
		panic(err.Error())
	}

	kClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	namespaceChannel := getNamespaceChannel(kClient)

	for {
		evt, ok := <-namespaceChannel
		if !ok {
			fmt.Print("APIServer ended our PVC watch, establishing a new watch.")
			namespaceChannel = getNamespaceChannel(kClient)
		}

		namespace, ok := evt.Object.(*v1.Namespace)

		if !ok {
			continue
		}

		if evt.Type != "ADDED" {
			continue
		}

		userAccountListOptions := metav1.ListOptions{
			LabelSelector: "ds-user=true",
		}

		userAccounts, err := kClient.CoreV1().ServiceAccounts("kube-system").List(userAccountListOptions)

		if err != nil {
			panic(err.Error())
		}

		for _, userAccount := range userAccounts.Items {
			userName, ok := userAccount.Annotations["ds-username"]

			if !ok {
				continue
			}

			if strings.HasPrefix(namespace.Name, userName) {

				roleBindingName := userName + "-" + namespace.Name

				_, err := kClient.RbacV1().RoleBindings(namespace.Name).Get(roleBindingName, metav1.GetOptions{})

				if err == nil {
					fmt.Printf("Existing RoleBinding for %v in namespace %v \n", userName, namespace.Name)
					continue
				}


				fmt.Printf("Creating RoleBinding for %v in namespace %v \n", userName, namespace.Name)

				roleBinding := RbacV1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: roleBindingName,
					},
					Subjects: []RbacV1.Subject{
						{
							Kind:      "ServiceAccount",
							Namespace: "kube-system",
							Name:      userAccount.Name,
						},
					},
					RoleRef: RbacV1.RoleRef{
						APIGroup: "rbac.authorization.k8s.io",
						Kind:     "ClusterRole",
						Name:     "admin",
					},
				}

				_, err = kClient.RbacV1().RoleBindings(namespace.Name).Create(&roleBinding)

				if err != nil {
					panic(err.Error())
				}

			}

		}

	}
}
