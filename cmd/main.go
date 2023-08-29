package main

import (
	"context"
	"flag"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var envString []string
	k8sContext := flag.String("context", "", "Name of the Azure Container Registry")
	deploymentName := flag.String("deployment", "", "Name of the repository in your registry")

	home := homedir.HomeDir()
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: filepath.Join(home, ".kube", "config")},
		&clientcmd.ConfigOverrides{
			CurrentContext: *k8sContext,
		}).ClientConfig()
	if err != nil {
		panic(err)
	}

	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err.Error())
	}

	deployments, err := clientSet.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})

	var parsedConfigMaps []string
	var parsedSecrets []string
	for _, deployment := range deployments.Items {
		if !strings.Contains(deployment.Name, *deploymentName) {
			continue
		}

		for _, container := range deployment.Spec.Template.Spec.Containers {
			if len(container.EnvFrom) == 0 && len(container.Env) == 0 {
				continue
			}

			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					parsedConfigMaps = append(parsedConfigMaps, envFrom.ConfigMapRef.Name)
				}

				if envFrom.SecretRef != nil {
					parsedSecrets = append(parsedSecrets, envFrom.SecretRef.Name)
				}
			}

			for _, env := range container.Env {
				envString = append(envString, fmt.Sprintf("%s=%s", env.Name, env.Value))
			}
		}
	}

	k8sSecrets, err := clientSet.CoreV1().Secrets("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	k8sConfigMaps, err := clientSet.CoreV1().ConfigMaps("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	// TODO: pretty sure this could be solved smarter with a generic type
	for _, parsedSecret := range parsedSecrets {
		for _, k8sSecret := range k8sSecrets.Items {
			if k8sSecret.Name != parsedSecret {
				continue
			}

			for key, value := range k8sSecret.Data {
				if len(value) == 0 {
					continue
				}

				envString = append(envString, fmt.Sprintf("%s=%s", key, string(value)))
			}
		}
	}

	for _, parsedConfigMap := range parsedConfigMaps {
		for _, k8sConfigMap := range k8sConfigMaps.Items {
			if k8sConfigMap.Name != parsedConfigMap {
				continue
			}

			for key, value := range k8sConfigMap.Data {
				if len(value) == 0 {
					continue
				}

				envString = append(envString, fmt.Sprintf("%s=%s", key, value))
			}
		}
	}

	// cleanup ";" for intellij configurations
	for index, es := range envString {
		if strings.Contains(es, ";") {
			envString[index] = strings.Replace(es, ";", "\\;", -1)
		}
	}

	envFilePath := ".env"
	envFile, err := os.Create(envFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer func(envFile *os.File) {
		err := envFile.Close()
		if err != nil {
			log.Fatal(err.Error())
		}
	}(envFile)

	for _, env := range envString {
		_, err := fmt.Fprintf(envFile, "%s\n", env)
		if err != nil {
			log.Fatal(err.Error())
		}
	}

	fmt.Printf("Environment variables written to %s\n", envFilePath)
}
