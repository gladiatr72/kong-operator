/*
Copyright (c) 2017, UPMC Enterprises
All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:
    * Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.
    * Neither the name UPMC Enterprises nor the
      names of its contributors may be used to endorse or promote products
      derived from this software without specific prior written permission.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL UPMC ENTERPRISES BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
*/

package pg

import (
	"github.com/Sirupsen/logrus"
	"github.com/upmc-enterprises/kong-operator/pkg/k8sutil"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/util/intstr"
)

// SimplePostgresDeployment returns a simple postgres deployment spec for testing purposes
func SimplePostgresDeployment(k *k8sutil.K8sutil, namespace string) error {
	var replicas int32
	replicas = int32(1)

	// Check if deployment exists
	deployment, err := k.Kclient.Deployments(namespace).Get("postgres")

	if len(deployment.Name) == 0 {
		logrus.Infof("%s not found, creating...", "postgres")

		deployment := &v1beta1.Deployment{
			ObjectMeta: v1.ObjectMeta{
				Name: "postgres",
				Labels: map[string]string{
					"app": "postgres",
				},
			},
			Spec: v1beta1.DeploymentSpec{
				Replicas: &replicas,
				Template: v1.PodTemplateSpec{
					ObjectMeta: v1.ObjectMeta{
						Labels: map[string]string{
							"app": "postgres",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							v1.Container{
								Name:            "postgres",
								Image:           "postgres:9.4",
								ImagePullPolicy: "Always",
								Env: []v1.EnvVar{
									v1.EnvVar{
										Name:  "POSTGRES_USER",
										Value: "kong",
									},
									v1.EnvVar{
										Name:  "POSTGRES_PASSWORD",
										Value: "kong",
									},
									v1.EnvVar{
										Name:  "POSTGRES_DB",
										Value: "kong",
									},
									v1.EnvVar{
										Name:  "PGDATA",
										Value: "/var/lib/postgresql/data/pgdata",
									},
								},
								Ports: []v1.ContainerPort{
									v1.ContainerPort{
										Name:          "postgres",
										ContainerPort: 5432,
										Protocol:      v1.ProtocolTCP,
									},
								},
								VolumeMounts: []v1.VolumeMount{
									v1.VolumeMount{
										Name:      "pg-data",
										MountPath: "/var/lib/postgresql/data",
									},
								},
							},
						},
						Volumes: []v1.Volume{
							v1.Volume{
								Name: "pg-data",
								VolumeSource: v1.VolumeSource{
									EmptyDir: &v1.EmptyDirVolumeSource{},
								},
							},
						},
					},
				},
			},
		}

		_, err := k.Kclient.Deployments(namespace).Create(deployment)

		if err != nil {
			logrus.Error("Could not create kong deployment: ", err)
			return err
		}
	} else if err != nil {
		logrus.Error("Could not get admin service: ", err)
		return err
	}

	return nil
}

// SimplePostgresService creates the postgres service
func SimplePostgresService(k *k8sutil.K8sutil, namespace string) error {

	// Check if service exists
	svc, err := k.Kclient.Services(namespace).Get("postgres")

	// Service missing, create
	if len(svc.Name) == 0 {
		logrus.Infof("%s not found, creating...", "postgres")

		clientSvc := &v1.Service{
			ObjectMeta: v1.ObjectMeta{
				Name: "postgres",
				Labels: map[string]string{
					"name": "postgres",
				},
			},
			Spec: v1.ServiceSpec{
				Selector: map[string]string{
					"app": "postgres",
				},
				Ports: []v1.ServicePort{
					v1.ServicePort{
						Name:       "pgql",
						Port:       5432,
						TargetPort: intstr.FromInt(5432),
						Protocol:   "TCP",
					},
				},
				Type: v1.ServiceTypeClusterIP,
			},
		}

		_, err := k.Kclient.Services(namespace).Create(clientSvc)

		if err != nil {
			logrus.Error("Could not create postgres service: ", err)
			return err
		}
	} else if err != nil {
		logrus.Error("Could not get postgres service: ", err)
		return err
	}

	return nil
}

// DeleteSimplePostgres cleans up deployment / service for postgres db
func DeleteSimplePostgres(k *k8sutil.K8sutil, namespace string) {

	err := k.Kclient.Services(namespace).Delete("postgres", &v1.DeleteOptions{})
	if err != nil {
		logrus.Error("Could not delete service postgres:", err)
	} else {
		logrus.Infof("Delete service: %s", "postgres")
	}

	err = k.Kclient.Services(namespace).Delete("postgres", &v1.DeleteOptions{})
	if err != nil {
		logrus.Error("Could not delete service postgres:", err)
	} else {
		logrus.Infof("Delete service: %s", "postgres")
	}

	// Get list of deployment
	deployment, err := k.Kclient.Deployments(namespace).Get("postgres")

	if err != nil {
		logrus.Error("Could not get deployments! ", err)
	}

	//Scale the deployment down to zero (https://github.com/kubernetes/client-go/issues/91)
	deployment.Spec.Replicas = new(int32)
	_, err = k.Kclient.Deployments(namespace).Update(deployment)

	if err != nil {
		logrus.Errorf("Could not scale deployment: %s ", deployment.Name)
	} else {
		logrus.Infof("Scaled deployment: %s to zero", deployment.Name)
	}

	err = k.Kclient.Deployments(namespace).Delete(deployment.Name, &v1.DeleteOptions{})

	if err != nil {
		logrus.Errorf("Could not delete deployments: %s ", deployment.Name)
	} else {
		logrus.Infof("Deleted deployment: %s", deployment.Name)
	}

	// Get list of ReplicaSets
	replicaSet, err := k.Kclient.ReplicaSets(namespace).Get("postgres")

	if err != nil {
		logrus.Error("Could not get replica sets! ", err)
	}

	err = k.Kclient.ReplicaSets(namespace).Delete(replicaSet.Name, &v1.DeleteOptions{})

	if err != nil {
		logrus.Errorf("Could not delete replica set: %s ", replicaSet.Name)
	} else {
		logrus.Infof("Deleted replica set: %s", replicaSet.Name)
	}
}
