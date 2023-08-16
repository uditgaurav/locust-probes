package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/uditgaurav/locust-probes/pkg/k8s"
	"github.com/uditgaurav/locust-probes/pkg/uploader"
	"github.com/uditgaurav/locust-probes/pkg/utils"

	"github.com/litmuschaos/litmus-go/pkg/log"
)

func main() {
	accountID := utils.Getenv("ACCOUNT_ID", "")
	projectID := utils.Getenv("PROJECT_ID", "")
	apiKey := utils.Getenv("API_KEY", "")
	chaosUID := utils.Getenv("CHAOS_UID", "")
	folderName := utils.Getenv("FOLDER_NAME", "chaos")

	podLogsMap, err := k8s.ExtractHelperPod(chaosUID)
	if err != nil {
		log.Errorf("error extracting logs from helper pod: %v", err)
		return
	}

	for podName, logs := range podLogsMap {
		rand.Seed(time.Now().UnixNano())
		randomNumber := rand.Int()

		log.Infof("pushing logs for pod: %v", podName)
		if err := uploader.PushToFileStore(accountID, projectID, apiKey, podName, logs, folderName, strconv.Itoa(randomNumber)); err != nil {
			log.Errorf("failed to push helper logs, err: %v", err)
			return
		}
	}
	fmt.Println("PASS")
}
