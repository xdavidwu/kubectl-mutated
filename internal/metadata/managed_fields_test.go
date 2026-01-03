package metadata

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	accorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	acmetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/structured-merge-diff/v6/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v6/value"
)

const (
	manualFieldManager  = "kubectl"
	machineFieldManager = "kustomize-controller"
)

func pods() corev1client.PodInterface {
	return client.CoreV1().Pods(corev1.NamespaceDefault)
}

func cleanupPods() {
	if err := pods().DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil {
		panic(err)
	}
}

func assertSetHasPath(t *testing.T, s *fieldpath.Set, parts ...any) {
	t.Helper()
	p := fieldpath.MakePathOrDie(parts...)
	if !s.Has(p) {
		t.Fatalf("set should have path %v", p)
	}
}

func assertSetNotHasPath(t *testing.T, s *fieldpath.Set, parts ...any) {
	t.Helper()
	p := fieldpath.MakePathOrDie(parts...)
	if s.Has(p) {
		t.Fatalf("set should have path %v", p)
	}
}

func makeSelectKV(k, v string) *value.FieldList {
	return &value.FieldList{{Name: k, Value: value.NewValueInterface(v)}}
}

func p[T any](v T) *T {
	return &v
}

func TestSoleyManuallyManagedSetFullyManual(t *testing.T) {
	t.Cleanup(cleanupPods)

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test",
					Image: "alpine:latest",
				},
			},
		},
	}

	created, err := pods().Create(t.Context(), &pod, metav1.CreateOptions{FieldManager: manualFieldManager})
	if err != nil {
		t.Fatalf("cannot create pod: %s", err)
	}

	set, err := SolelyManuallyManagedSet(created.ManagedFields)
	if err != nil {
		t.Fatalf("cannot find solely manually managed set: %s", set)
	}
	assertSetHasPath(t, set, "spec", "containers", makeSelectKV("name", "test"), "name")
	assertSetHasPath(t, set, "spec", "containers", makeSelectKV("name", "test"), "image")
}

func TestSoleyManuallyManagedSetFullyMachine(t *testing.T) {
	t.Cleanup(cleanupPods)

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test",
					Image: "alpine:latest",
				},
			},
		},
	}

	created, err := pods().Create(t.Context(), &pod, metav1.CreateOptions{FieldManager: machineFieldManager})
	if err != nil {
		t.Fatalf("cannot create pod: %s", err)
	}

	set, err := SolelyManuallyManagedSet(created.ManagedFields)
	if err != nil {
		t.Fatalf("cannot find solely manually managed set: %s", set)
	}
	assertSetNotHasPath(t, set, "spec", "containers", makeSelectKV("name", "test"), "name")
	assertSetNotHasPath(t, set, "spec", "containers", makeSelectKV("name", "test"), "image")
}

func TestSoleyManuallyManagedSetCoManaged(t *testing.T) {
	t.Cleanup(cleanupPods)

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test",
					Image: "alpine:latest",
				},
			},
		},
	}

	_, err := pods().Create(t.Context(), &pod, metav1.CreateOptions{FieldManager: machineFieldManager})
	if err != nil {
		t.Fatalf("cannot create pod: %s", err)
	}

	patch := accorev1.PodApplyConfiguration{
		TypeMetaApplyConfiguration: acmetav1.TypeMetaApplyConfiguration{
			APIVersion: p("v1"),
			Kind:       p("Pod"),
		},
		ObjectMetaApplyConfiguration: &acmetav1.ObjectMetaApplyConfiguration{
			Name: p("test"),
		},
		Spec: &accorev1.PodSpecApplyConfiguration{
			Containers: []accorev1.ContainerApplyConfiguration{
				{
					Name:  p("test"),
					Image: p("alpine:latest"),
				},
			},
		},
	}
	applied, err := pods().Apply(t.Context(), &patch, metav1.ApplyOptions{FieldManager: manualFieldManager})
	if err != nil {
		t.Fatalf("cannot apply pod: %s", err)
	}

	set, err := SolelyManuallyManagedSet(applied.ManagedFields)
	if err != nil {
		t.Fatalf("cannot find solely manually managed set: %s", set)
	}
	assertSetNotHasPath(t, set, "spec", "containers", makeSelectKV("name", "test"), "name")
	assertSetNotHasPath(t, set, "spec", "containers", makeSelectKV("name", "test"), "image")
}

func TestSoleyManuallyManagedSetMixed(t *testing.T) {
	t.Cleanup(cleanupPods)

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "test",
					Image:           "alpine:latest",
					ImagePullPolicy: corev1.PullAlways,
				},
			},
		},
	}

	_, err := pods().Create(t.Context(), &pod, metav1.CreateOptions{FieldManager: machineFieldManager})
	if err != nil {
		t.Fatalf("cannot create pod: %s", err)
	}

	patch := accorev1.PodApplyConfiguration{
		TypeMetaApplyConfiguration: acmetav1.TypeMetaApplyConfiguration{
			APIVersion: p("v1"),
			Kind:       p("Pod"),
		},
		ObjectMetaApplyConfiguration: &acmetav1.ObjectMetaApplyConfiguration{
			Name: p("test"),
		},
		Spec: &accorev1.PodSpecApplyConfiguration{
			Containers: []accorev1.ContainerApplyConfiguration{
				{
					Name:  p("test"),
					Image: p("alpine:3.22"),
				},
			},
		},
	}
	applied, err := pods().Apply(t.Context(), &patch, metav1.ApplyOptions{Force: true, FieldManager: manualFieldManager})
	if err != nil {
		t.Fatalf("cannot apply pod: %s", err)
	}

	set, err := SolelyManuallyManagedSet(applied.ManagedFields)
	if err != nil {
		t.Fatalf("cannot find solely manually managed set: %s", set)
	}
	assertSetHasPath(t, set, "spec", "containers", makeSelectKV("name", "test"), "image")
	assertSetNotHasPath(t, set, "spec", "containers", makeSelectKV("name", "test"), "imagePullPolicy")
}
