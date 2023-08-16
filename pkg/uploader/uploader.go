package uploader

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/pkg/errors"
)

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
