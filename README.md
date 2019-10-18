# Logstash Operator

This repo is a fork of [Elastic Cloud on Kubernetes (ECK)](https://github.com/elastic/cloud-on-k8s)
with support for Logstash.

## Install

Install the operator and CRDs:

```shell
kubectl apply -f https://raw.githubusercontent.com/cloudptio/logstash-operator/master/config/all-in-one-flavor-default.yaml
```

*Or locally:*

```shell
kubectl apply -f config/all-in-one-flavor-default.yaml
```

Monitor the operator logs:

```shell
kubectl -n elastic-system logs -f statefulset.apps/elastic-operator
```

## Use

Deploy an Elasticsearch cluster:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: elasticsearch.k8s.elastic.co/v1beta1
kind: Elasticsearch
metadata:
  name: quickstart
spec:
  version: 7.4.0
  nodeSets:
  - name: default
    count: 1
    config:
      node.master: true
      node.data: true
      node.ingest: true
      node.store.allow_mmap: false
EOF
```

Deploy Logstash:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: logstash.k8s.elastic.co/v1beta1
kind: Logstash
metadata:
  name: quickstart
spec:
  version: 7.4.0
  count: 1
  elasticsearchRef:
    name: quickstart
  inputConf: |
    input {
      beats {
        port => 5044
      }
    }
EOF
```

Wait for Logstash to come up:

```shell
kubectl get logstash quickstart -w
```

## An Example with Filebeat and Kibana

Ingest all the K8S Pod logs with Filebeat:

```shell
# NOTE: you'll need to swap `quickstart` with your cluster name.
kubectl apply -f config/samples/filebeat/filebeat.yaml
```

Setup Kibana:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: kibana.k8s.elastic.co/v1beta1
kind: Kibana
metadata:
  name: quickstart
spec:
  version: 7.4.0
  count: 1
  elasticsearchRef:
    name: quickstart
EOF
```

Wait for Kibana to come up:

```shell
kubectl get kibana quickstart -w
```

Expose the Kibana port locally to view data:

```shell
kubectl port-forward service/quickstart-kb-http 5601
```

And finally, go to [https://localhost:5601](https://localhost:5601).

To view your Kibana login info:

```shell
# Username is elastic
# Password is:
kubectl get secret quickstart-es-elastic-user -o=jsonpath='{.data.elastic}' | base64 --decode
```

---

You can follow the
[the official getting started guide](https://www.elastic.co/guide/en/cloud-on-k8s/current/k8s-quickstart.html)
and
view the [samples/](config/samples/) to see more on how to configure each of the ELK
components.
