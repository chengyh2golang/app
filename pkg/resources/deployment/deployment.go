package deployment

import (
	appv1alpha1 "app/pkg/apis/app/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func New(app *appv1alpha1.App) appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta:metav1.TypeMeta{
			APIVersion:"apps/v1",
			Kind:"Deployment",
		},
		ObjectMeta:metav1.ObjectMeta{
			Namespace:app.Namespace,
			Name:app.Name,
		},
		Spec:appsv1.DeploymentSpec{
			Replicas:app.Spec.Replicas,
			Template:corev1.PodTemplateSpec{
				Spec:corev1.PodSpec{
					Containers:newContainers(app),

				},
			},

		},
	}
}

func newContainers(app *appv1alpha1.App) []corev1.Container {
	containerPort := []corev1.ContainerPort{}
	for _,servicePort := range app.Spec.Ports {
		cport := corev1.ContainerPort{}
		cport.ContainerPort = servicePort.TargetPort.IntVal
		containerPort = append(containerPort,cport)
	}
	return []corev1.Container{
		{
			Name:app.Name,
			Image:app.Spec.Image,
			//Args:[]string{"--bind_ip_all","--replSet=rs0","--keyFile=/etc/mongo/default-key"},
			Resources:app.Spec.Resources,
			Ports:containerPort,
			Env:app.Spec.Envs,
		},
	}
}