## Running `honeycomb-opentracing-proxy` in Kubernetes

This directory contains an example manifest for deploying
`honeycomb-opentracing-proxy` in a Kubernetes cluster.

First, save your Honeycomb write key as a Kubernetes secret.

```
kubectl create secret generic -n default honeycomb-writekey --from-literal=key=$YOUR_WRITE_KEY
```

Then create a Deployment and Service for the proxy:

```
kubectl apply -f example-manifest.yaml
```

This will expose the proxy inside your cluster at the address

```
honeycomb-opentracing-proxy.default:9411
```
