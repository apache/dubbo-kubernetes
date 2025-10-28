package inject

import corev1 "k8s.io/api/core/v1"

func FindProxy(pod *corev1.Pod) *corev1.Container {
	return FindContainerFromPod(ProxyContainerName, pod)
}

func FindContainerFromPod(name string, pod *corev1.Pod) *corev1.Container {
	if c := FindContainer(name, pod.Spec.Containers); c != nil {
		return c
	}
	return FindContainer(name, pod.Spec.InitContainers)
}
