# katib-experiment-webhook

katib hyperparameter tuning 과정에서 istio sidecar로 인해 job이 complete되지 못하는 현상을 해당 pod template에 "sidecar.istio.io/inject": "false" annotation을 달아줌으로써 해결하는 웹훅
