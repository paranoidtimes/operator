Example of having a deployment restart when an associated configmap is edited. NOTE see TODO section for known bugs and needed improvements.

Build with

```
operator-sdk init --domain github.com --repo github.com/paranoidtimes/operator
operator-sdk create api --group batch --kind ConfigMapWatcher --version v1 --resource true --controller true
```
copy both controllers/configmapwatcher_controller.go and api/v1/configmapwatcher_types.go into same place in directory structure

```
make install
```
update IMG in Makefile to match docker registry (paranoidtimes/operator:01 in this case)

```
make docker-build docker-push
kubectl apply -f example/clusterrole.yaml # apply cluster role (will likely need editing to match domain)
kubectl create clusterrolebinding default-view --clusterrole=cmwatcheroperator --serviceaccount=default:default
kubectl run --rm -i demo --image=paranoidtimes/operator:01
```

Deploy a ConfigMap (cm.yaml) a deployment (deploy.yaml) and a ConfigMapWatcher (watcher.yaml)

TODO:
* Currently there is no error checking on the configmap or deployment labels of the ConfigMapWatcher, this will result in errors if they do not exist. As noted in code.
* Additionally will overwrite the initial env var, needs to scan env vars and update or add the correct one.
* Currently only runs in default namespace. Add code to allow to run in other namespaces
