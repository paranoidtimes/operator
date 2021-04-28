/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "github.com/paranoidtimes/operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"crypto/sha1"
	"encoding/base64"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ConfigMapWatcherReconciler reconciles a ConfigMapWatcher object
type ConfigMapWatcherReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=batch.github.com,resources=configmapwatchers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch.github.com,resources=configmapwatchers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=batch.github.com,resources=configmapwatchers/finalizers,verbs=update

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *ConfigMapWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("configmapwatcher", req.NamespacedName)
	// get configmap watcher info
	var watcher batchv1.ConfigMapWatcher
	if err := r.Get(ctx, req.NamespacedName, &watcher); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	var configmaps corev1.ConfigMapList
	if err := r.List(ctx, &configmaps); err != nil {
		return ctrl.Result{}, err
	}

	// Create a CoreV1Client (k8s.io/client-go/kubernetes/typed/core/v1)
	coreV1Client := clientset.CoreV1()

    // TODO error checking for if the configmap actually exists
	configmap, err := coreV1Client.ConfigMaps("default").Get(context.TODO(), watcher.Spec.ConfigMap, metav1.GetOptions{})

    // make a hash of the config map, use this to update deployment
	var sha string
	hash := sha1.New()
	hash.Write([]byte(configmap.Data["config.cfg"]))
	sha = base64.URLEncoding.EncodeToString(hash.Sum(nil))

	var deployments appsv1.DeploymentList
	if err := r.List(ctx, &deployments); err != nil {
		return ctrl.Result{}, err
	}

	deploymentsClient := clientset.AppsV1().Deployments(apiv1.NamespaceDefault)

    // TODO error checking for if the deployment actually exists
	result, getErr := deploymentsClient.Get(context.TODO(), watcher.Spec.Deployment, metav1.GetOptions{})
	if getErr != nil {
		panic(fmt.Errorf("Failed to get Deployment: %v", getErr))
	}

	// TODO Currently this will overwrite the first variable. This needs to be updated to examine the current variables, and update "config_hash" if it finds it, or add it to the end if it does not.
    // update an env var in the deployment. This method allows for the deployment to track what state it is in, solving the issue of "did the config map change", rather than identifying that, just update the deployment, if it is the same value nothing happens, if it is updated, the deployment will reroll.
	result.Spec.Template.Spec.Containers[0].Env[0].Value = sha
	_, updateErr := deploymentsClient.Update(context.TODO(), result, metav1.UpdateOptions{})
	if updateErr != nil {
		panic(fmt.Errorf("Failed to update! %v\n", updateErr))
	}

	return ctrl.Result{RequeueAfter: time.Duration(watcher.Spec.Frequency) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.ConfigMapWatcher{}).
		Complete(r)
}
