package deployment

import (
	appv1alpha1 "app/pkg/apis/app/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func New(app *appv1alpha1.App) *appsv1.Deployment {
	labels := map[string]string{
		"app.example.com/v1alpha1":app.Name,
	}
	selector := &metav1.LabelSelector{
		MatchLabels:labels,
	}
	return &appsv1.Deployment{
		TypeMeta:metav1.TypeMeta{
			APIVersion:"apps/v1",
			Kind:"Deployment",
		},
		ObjectMeta:metav1.ObjectMeta{
			Namespace:app.Namespace,
			Name:app.Name,
			Labels:labels,
			OwnerReferences:[]metav1.OwnerReference{
				*metav1.NewControllerRef(app, schema.GroupVersionKind{
					Group:appv1alpha1.SchemeGroupVersion.Group,
					Version:appv1alpha1.SchemeGroupVersion.Version,
					Kind:"App",
				}),
			},
		},
		Spec:appsv1.DeploymentSpec{
			Selector:selector,
			Replicas:app.Spec.Replicas,
			Template:corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:labels,
				},
				Spec:corev1.PodSpec{
					Containers:newContainers(app),

				},
			},

		},
	}
}

func newContainers(app *appv1alpha1.App) []corev1.Container {
	var containerPort []corev1.ContainerPort
	for _,servicePort := range app.Spec.Ports {
		cPort := corev1.ContainerPort{}
		cPort.ContainerPort = servicePort.TargetPort.IntVal
		containerPort = append(containerPort,cPort)
	}
	return []corev1.Container{
		{
			Name:app.Name,
			Image:app.Spec.Image,
			ImagePullPolicy:corev1.PullIfNotPresent,
			//Args:[]string{"--bind_ip_all","--replSet=rs0","--keyFile=/etc/mongo/default-key"},
			Resources:app.Spec.Resources,
			Ports:containerPort,
			Env:app.Spec.Envs,
		},
	}
}