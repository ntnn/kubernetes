package namespace

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNamespaceController_Shutdown(t *testing.T) {

	// Mock discoverResourcesFn to return a list of resources, without
	// this the NamespacedResourcesDeleter will call os.Exit in
	// initOpCache as there are no applicable resources.
	discoverResourcesFn := func() ([]*metav1.APIResourceList, error) {
		return []*metav1.APIResourceList{
			{
				GroupVersion: "v1",
				APIResources: []metav1.APIResource{
					{
						Name:       "pods",
						Namespaced: true,
						Kind:       "Pod",
						Verbs:      []string{"get", "list", "delete", "deletecollection", "create", "update"},
					},
				},
			},
		}, nil
	}

	fakeClientset := fake.NewSimpleClientset()
	informerFactory := informers.NewSharedInformerFactory(fakeClientset, 0)

	stopCh := make(chan struct{})
	defer close(stopCh)

	// TODO cleanup
	informerFactory.Start(stopCh)
	namespaceInformer := informerFactory.Core().V1().Namespaces()
	go namespaceInformer.Informer().Run(stopCh)
	informerFactory.WaitForCacheSync(stopCh)
	for !namespaceInformer.Informer().HasSynced() {
		time.Sleep(100 * time.Millisecond)
	}

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// TODO k8s testing context
	ctx := context.Background()

	nm, err := NewNamespaceController(
		ctx,
		fakeClientset,
		nil, // metadata.Interface
		discoverResourcesFn,
		namespaceInformer,
		0,
		v1.FinalizerKubernetes,
	)
	if err != nil {
		t.Errorf("failed to create namespace controller: %v", err)
	}
	nm.Shutdown()
}
