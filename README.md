# Helper Pod Logs Extractor

This app is designed to extract logs from helper pods in a Kubernetes cluster, and then push these logs to the Harness file store. The app retrieves the helper pod logs by filtering with a provided `CHAOS_UID` and pushes each pod's logs to the Harness file store. 

## Usage

1. Set up the necessary environment variables.
2. Run the application in a probe as shown below.

```yaml
- name: logs
  type: cmdProbe
  mode: EOT
  runProperties:
    probeTimeout: 180s
    retry: 0
    interval: 1s
    stopOnFailure: true
  cmdProbe/inputs:
    command: ./app
    source:
      image: docker.io/chaosnative/locust:0.1.0
      inheritInputs: true
    comparator:
      type: string
      criteria: contains
      value: PASS
```

## Configuration

The application relies on various environment variables. Here's a table outlining each variable, its default value, and its usage:

| Variable   | Default Value | Usage                                                                     |
|------------|---------------|---------------------------------------------------------------------------|
| `ACCOUNT_ID`   | ""            | The account ID used for pushing the logs.                                |
| `PROJECT_ID`   | ""            | The project ID under which the logs will be stored.                      |
| `API_KEY`      | ""            | The API key to authenticate the request for pushing logs.                |
| `CHAOS_UID`    | ""            | The unique identifier to fetch logs from pods with matching `chaosUID`.  |
| `FOLDER_NAME`  | "chaos"       | The name of the folder where logs will be stored in the file store.      |
| `KUBECONFIG`   | *system-defined*  | Path to a kubeconfig. Only required if out-of-cluster.                   |

Remember to always set sensitive information like `API_KEY` securely and avoid exposing it in plain text or logs.