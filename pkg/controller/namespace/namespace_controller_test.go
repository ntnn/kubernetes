package namespace

import (
	"testing"
	"time"

	"go.uber.org/goleak"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/kubernetes/test/utils/ktesting"
)

func TestNamespaceController_Shutdown(t *testing.T) {
	_, tCtx := ktesting.NewTestContext(t)

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

	informerFactory.Start(tCtx.Done())
	namespaceInformer := informerFactory.Core().V1().Namespaces()
	go namespaceInformer.Informer().Run(tCtx.Done())
	informerFactory.WaitForCacheSync(tCtx.Done())
	for !namespaceInformer.Informer().HasSynced() {
		time.Sleep(100 * time.Millisecond)
	}

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	nm, err := NewNamespaceController(
		tCtx,
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
