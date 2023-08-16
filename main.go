package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/litmuschaos/litmus-go/pkg/log"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	accountID := Getenv("ACCOUNT_ID", "")
	projectID := Getenv("PROJECT_ID", "")
	apiKey := Getenv("API_KEY", "")
	chaosUID := Getenv("CHAOS_UID", "")
	folderName := Getenv("FOLDER_NAME", "chaos")

	podLogsMap, err := extractHelperPod(chaosUID)
	if err != nil {
		log.Errorf("error extracting logs from helper pod: %v", err)
		return
	}

	for podName, logs := range podLogsMap {
		rand.Seed(time.Now().UnixNano())
		randomNumber := rand.Int()

		log.Infof("pushing logs for pod: %v", podName)
		if err := PushToFileStore(accountID, projectID, apiKey, podName, logs, folderName, strconv.Itoa(randomNumber)); err != nil {
			log.Errorf("failed to push helper logs, err: %v", err)
			return
		}
	}
	fmt.Println("PASS")
}

func extractHelperPod(chaosUID string) (map[string]string, error) {

	kubeconfig := os.Getenv("KUBECONFIG")

	var config *rest.Config
	var err error

	if kubeconfig == "" {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get in-cluster configuration")
		}
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to build config from KUBECONFIG: %s", kubeconfig)
		}
	}

	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("chaosUID=%s", chaosUID),
	})
	if err != nil {
		return nil, err
	}

	podLogsMap := make(map[string]string)

	for _, pod := range pods.Items {
		if strings.Contains(pod.Name, "helper") {
			podLogOpts := corev1.PodLogOptions{}
			req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get logs for pod: %s", pod.Name)
			}
			defer podLogs.Close()

			buf := new(strings.Builder)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				return nil, err
			}

			podLogsMap[pod.Name] = buf.String()
		}
	}

	if len(podLogsMap) == 0 {
		return nil, errors.Errorf("No matching pods found")
	}

	return podLogsMap, nil
}

// PushToFileStore will push the file content in the File Store
func PushToFileStore(accountID, projectID, apiKey, podName, logs, folderName, randomNumber string) error {
	client := &http.Client{}

	identifier := "chaos_test_txt_" + randomNumber

	formattedData := fmt.Sprintf(`------WebKitFormBoundaryxzJwMPPeTzodmhMO
Content-Disposition: form-data; name="type"

FILE
------WebKitFormBoundaryxzJwMPPeTzodmhMO
Content-Disposition: form-data; name="content"; filename="%s"
Content-Type: text/plain

%s

------WebKitFormBoundaryxzJwMPPeTzodmhMO
Content-Disposition: form-data; name="mimeType"

txt
------WebKitFormBoundaryxzJwMPPeTzodmhMO
Content-Disposition: form-data; name="name"

%s
------WebKitFormBoundaryxzJwMPPeTzodmhMO
Content-Disposition: form-data; name="identifier"

%s
------WebKitFormBoundaryxzJwMPPeTzodmhMO
Content-Disposition: form-data; name="parentIdentifier"

%s
------WebKitFormBoundaryxzJwMPPeTzodmhMO--`, podName, logs, podName, identifier, folderName)

	data := strings.NewReader(formattedData)

	req, err := http.NewRequest("POST", fmt.Sprintf("https://app.harness.io/gateway/ng/api/file-store?routingId=%s&accountIdentifier=%s&orgIdentifier=default&projectIdentifier=%s", accountID, accountID, projectID), data)
	if err != nil {
		return errors.Wrap(err, "failed to create new request")
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("content-type", "multipart/form-data; boundary=----WebKitFormBoundaryxzJwMPPeTzodmhMO")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)
		return errors.Errorf("failed to push logs for %v pod, status code: %v, response: %s", podName, resp.StatusCode, bodyString)
	}

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Info("File created successfully")

	return nil
}

func Getenv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
