package util

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateNamespace(cs kubernetes.Interface, namespace string, dryRun bool) error {
	if dryRun {
		return nil
	}
	if namespace == "" {
		namespace = "dubbo-system"
	}
	if _, err := cs.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   namespace,
				Labels: map[string]string{},
			}}
			_, err := cs.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create namespace %v: %v", namespace, err)
			}

			return nil
		}

		return fmt.Errorf("failed to check if namespace %v exists: %v", namespace, err)
	}

	return nil
}
