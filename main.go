package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var apiKey string
var accountIdentifier string
var orgIdentifier string
var projectIdentifier string

func main() {
	var config *rest.Config
	var err error

	// Define the flags
	flag.StringVar(&apiKey, "api-key", "", "Harness API key")
	flag.StringVar(&accountIdentifier, "account-identifier", "", "Account Identifier")
	flag.StringVar(&orgIdentifier, "org-identifier", "", "Organization Identifier")
	flag.StringVar(&projectIdentifier, "project-identifier", "", "Project Identifier")

	// Parse the flags
	flag.Parse()

	// Get KUBECONFIG from environment variable
	kubeconfig := os.Getenv("KUBECONFIG")

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// Use in-cluster configuration
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// Get CHAOS_UID from the environment variable
	chaosUIDEnv := os.Getenv("CHAOS_UID")

	// Create a context
	ctx := context.TODO()

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{}) // Corrected line
	if err != nil {
		panic(err)
	}

	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "helper") {
			if chaosUIDLabel, found := pod.Labels["chaosUID"]; found && chaosUIDLabel == chaosUIDEnv {
				logOptions := v1.PodLogOptions{}
				req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &logOptions)
				podLogs, err := req.DoRaw(ctx)
				if err != nil {
					panic(err)
				}

				err = uploadToHarness(pod.Name, podLogs)
				if err != nil {
					fmt.Println("Failed to upload logs:", err)
				}

				fmt.Println("PASS")
			}
		}
	}
}

func uploadToHarness(podName string, logContent []byte) error {
	url := fmt.Sprintf("https://app.harness.io/ng/api/file-store?accountIdentifier=%s&orgIdentifier=%s&projectIdentifier=%s", accountIdentifier, orgIdentifier, projectIdentifier)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add required fields
	writer.WriteField("name", podName)
	writer.WriteField("fileUsage", "MANIFEST_FILE")
	writer.WriteField("type", "FILE")
	writer.WriteField("path", "chaos/"+podName) // Path to the folder named chaos

	// Add log content
	contentPart, err := writer.CreateFormField("content")
	if err != nil {
		return err
	}
	contentPart.Write(logContent)

	err = writer.Close()
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	// Add API key header (from flag)
	req.Header.Add("x-api-key", apiKey)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to upload logs, status code: %d", resp.StatusCode)
	}

	return nil
}
