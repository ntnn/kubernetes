package namespace

import (
	"context"
	"testing"

	"go.uber.org/goleak"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNamespaceController_Shutdown(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	fakeClientset := fake.NewSimpleClientset()
	informers.NewSharedInformerFactory(fakeClientset, 0)

	// TODO k8s testing context
	ctx := context.Background()

	nm := NewNamespaceController(ctx, fakeClientset, nil, nil, nil, nil, nil)
	nm.Shutdown()
}
