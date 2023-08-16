package k8s

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ExtractHelperPod(chaosUID string) (map[string]string, error) {

	clientset, err := ClientSet()
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
