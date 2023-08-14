# locust-probes


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
                                image: docker.io/uditgaurav/locust:latest
                                inheritInputs: true
                              comparator:
                                type: string
                                criteria: contains
                                value: PASS
```