package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
	var helperPodFound bool

	totalChaosDurationEnv, err := strconv.Atoi(os.Getenv("TOTAL_CHAOS_DURATION"))
	if err != nil {
		log.Fatalf("Error parsing TOTAL_CHAOS_DURATION: %v", err)
	}

	time.Sleep(time.Duration(totalChaosDurationEnv) * time.Second)

	flag.StringVar(&apiKey, "api-key", "", "Harness API key")
	flag.StringVar(&accountIdentifier, "account-identifier", "", "Account Identifier")
	flag.StringVar(&orgIdentifier, "org-identifier", "", "Organization Identifier")
	flag.StringVar(&projectIdentifier, "project-identifier", "", "Project Identifier")

	flag.Parse()

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		log.Fatalf("Failed to configure k8s client: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create k8s clientset: %v", err)
	}

	chaosUIDEnv := os.Getenv("CHAOS_UID")
	ctx := context.TODO()

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to list pods: %v", err)
	}

	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "helper") {
			if chaosUIDLabel, found := pod.Labels["chaosUID"]; found && chaosUIDLabel == chaosUIDEnv {
				helperPodFound = true
				logOptions := v1.PodLogOptions{}
				req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &logOptions)
				podLogs, err := req.DoRaw(ctx)
				if err != nil {
					log.Printf("Failed to get logs for pod %s: %v", pod.Name, err)
					continue
				}

				err = uploadToHarness(pod.Name, podLogs)
				if err != nil {
					log.Printf("Failed to upload logs for pod %s: %v", pod.Name, err)
				}

				fmt.Println("PASS")
			}
		}
	}

	if !helperPodFound {
		log.Println("ERROR: Helper pod not found.")
	}
}

func uploadToHarness(podName string, logContent []byte) error {
	url := fmt.Sprintf("https://app.harness.io/ng/api/file-store?accountIdentifier=%s&orgIdentifier=%s&projectIdentifier=%s", accountIdentifier, orgIdentifier, projectIdentifier)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("name", podName); err != nil {
		return err
	}

	if err := writer.WriteField("fileUsage", "MANIFEST_FILE"); err != nil {
		return err
	}

	if err := writer.WriteField("type", "FILE"); err != nil {
		return err
	}

	if err := writer.WriteField("path", "chaos/"+podName); err != nil {
		return err
	}

	contentPart, err := writer.CreateFormField("content")
	if err != nil {
		return err
	}

	if _, err = contentPart.Write(logContent); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

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
