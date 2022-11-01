package kube

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	greptimev1alpha1 "github.com/GreptimeTeam/gtctl/third_party/apis/v1alpha1"
)

type Client struct {
	kubeClient        kubernetes.Interface
	dynamicKubeClient dynamic.Interface
	discoveryClient   discovery.DiscoveryInterface
}

var addToScheme sync.Once

// FIXME(zyy17): Do we have the more elegant way to get GVR of GreptimeDBCluster?
var greptimeDBClusterGVR = schema.GroupVersionResource{
	Group:    "greptime.io",
	Version:  "v1alpha1",
	Resource: "greptimedbclusters",
}

func NewClient(kubeconfig string) (*Client, error) {
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			return nil, fmt.Errorf("kubeconfig not found")
		}
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamicKubeClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	// Add CRDs to the scheme. They are missing by default.
	addToScheme.Do(func() {
		if err := apiextensionsv1.AddToScheme(scheme.Scheme); err != nil {
			// This should never happen.
			panic(err)
		}
		if err := apiextensionsv1beta1.AddToScheme(scheme.Scheme); err != nil {
			panic(err)
		}
		if err := greptimev1alpha1.AddToScheme(scheme.Scheme); err != nil {
			panic(err)
		}
	})

	return &Client{
		kubeClient:        kubeClient,
		dynamicKubeClient: dynamicKubeClient,
		discoveryClient:   discoveryClient,
	}, nil
}

func (c *Client) Apply(manifests []byte) error {
	builder := resource.NewLocalBuilder().
		// Configure with a scheme to get typed objects in the versions registered with the scheme.
		// As an alternative, could call Unstructured() to get unstructured objects.
		WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
		// Provide input via a Reader.
		// As an alternative, could call Path(false, "/path/to/file") to read from a file.
		Stream(bytes.NewBufferString(string(manifests)), "input").
		// Flatten items contained in List objects
		Flatten().
		// Accumulate as many items as possible
		ContinueOnError()

	// Run the builder
	result := builder.Do()

	if err := result.Err(); err != nil {
		return err
	}

	items, err := result.Infos()
	if err != nil {
		return err
	}

	isNamespaced, err := c.isNamespacedResource()
	if err != nil {
		return err
	}

	for _, item := range items {
		unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(item.Object)
		if err != nil {
			return err
		}
		gvk := item.Object.GetObjectKind().GroupVersionKind()

		gvr := schema.GroupVersionResource{
			Group:   gvk.Group,
			Version: gvk.Version,
			// FIXME(zyy17): Maybe some resources don't have plural.
			Resource: strings.ToLower(gvk.Kind) + "s",
		}

		ctx := context.TODO()

		if isNamespaced[gvr.Resource] {
			var ns string
			if item.Namespace == "" {
				ns = "default"
			} else {
				ns = item.Namespace
			}
			_, err = c.dynamicKubeClient.Resource(gvr).Namespace(ns).Apply(ctx, item.Name,
				&unstructured.Unstructured{Object: unstructuredObj},
				metav1.ApplyOptions{FieldManager: "application/apply-patch"})
			if err != nil {
				return err
			}
		} else {
			_, err = c.dynamicKubeClient.Resource(gvr).Apply(ctx, item.Name,
				&unstructured.Unstructured{Object: unstructuredObj},
				metav1.ApplyOptions{FieldManager: "application/apply-patch"})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Client) GetCluster(ctx context.Context, name, namespace string) (*greptimev1alpha1.GreptimeDBCluster, error) {
	return c.getCluster(ctx, name, namespace)
}

func (c *Client) GetAllClusters(ctx context.Context) (*greptimev1alpha1.GreptimeDBClusterList, error) {
	return c.getAllClusters(ctx)
}

func (c *Client) DeleteCluster(ctx context.Context, name, namespace string) error {
	return c.dynamicKubeClient.Resource(greptimeDBClusterGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Client) UpdateCluster(ctx context.Context, namespace string, cluster *greptimev1alpha1.GreptimeDBCluster) error {
	unstructuredObject, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cluster)
	if err != nil {
		return err
	}

	_, err = c.dynamicKubeClient.Resource(greptimeDBClusterGVR).Namespace(namespace).Update(ctx, &unstructured.Unstructured{Object: unstructuredObject}, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteEtcdCluster(ctx context.Context, name, namespace string) error {
	if err := c.kubeClient.CoreV1().Services(namespace).Delete(ctx, fmt.Sprintf("%s-%s", name, "etcd-svc"), metav1.DeleteOptions{}); err != nil {
		return err
	}

	if err := c.kubeClient.AppsV1().StatefulSets(namespace).Delete(ctx, fmt.Sprintf("%s-%s", name, "etcd"), metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *Client) WaitForDeploymentReady(name, namespace string, timeout time.Duration) error {
	conditionFunc := func() (bool, error) {
		return c.isDeploymentReady(context.TODO(), name, namespace)
	}

	if int(timeout) < 0 {
		return wait.PollInfinite(time.Second, conditionFunc)
	} else {
		return wait.PollImmediate(time.Second, timeout, conditionFunc)
	}
}

func (c *Client) WaitForClusterReady(name, namespace string, timeout time.Duration) error {
	conditionFunc := func() (bool, error) {
		return c.isClusterReady(context.TODO(), name, namespace)
	}

	if int(timeout) < 0 {
		return wait.PollInfinite(time.Second, conditionFunc)
	} else {
		return wait.PollImmediate(time.Second, timeout, conditionFunc)
	}
}

func (c *Client) isDeploymentReady(ctx context.Context, name, namespace string) (bool, error) {
	deployment, err := c.kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable &&
			condition.Status == corev1.ConditionTrue {
			return true, nil
		}
	}

	return false, nil
}

func (c *Client) isClusterReady(ctx context.Context, name, namespace string) (bool, error) {
	cluster, err := c.getCluster(ctx, name, namespace)
	if err != nil {
		return false, nil
	}

	for _, condition := range cluster.Status.Conditions {
		if condition.Type == greptimev1alpha1.GreptimeDBClusterReady &&
			condition.Status == corev1.ConditionTrue {
			return true, nil
		}
	}

	return false, nil
}

// FIXME(zyy17): Generate clientset for Greptime CRDs.

func (c *Client) getCluster(ctx context.Context, name, namespace string) (*greptimev1alpha1.GreptimeDBCluster, error) {
	unstructuredObject, err := c.dynamicKubeClient.Resource(greptimeDBClusterGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var cluster greptimev1alpha1.GreptimeDBCluster
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.UnstructuredContent(), &cluster); err != nil {
		return nil, err
	}

	return &cluster, nil
}

func (c *Client) getAllClusters(ctx context.Context) (*greptimev1alpha1.GreptimeDBClusterList, error) {
	unstructuredObject, err := c.dynamicKubeClient.Resource(greptimeDBClusterGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var clusters greptimev1alpha1.GreptimeDBClusterList
	if err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObject.UnstructuredContent(), &clusters); err != nil {
		return nil, err
	}

	return &clusters, nil
}

func (c *Client) isNamespacedResource() (map[string]bool, error) {
	// How to get the list: kubectl api-resources --namespaced | grep -v NAME | awk '{print "\""$1"\"""\,"}'.
	isNamespaced := make(map[string]bool)
	_, apiResourcesList, err := c.discoveryClient.ServerGroupsAndResources()
	if err != nil {
		return isNamespaced, err
	}
	for _, list := range apiResourcesList {
		for _, r := range list.APIResources {
			isNamespaced[r.Name] = r.Namespaced
		}
	}
	return isNamespaced, nil
}
