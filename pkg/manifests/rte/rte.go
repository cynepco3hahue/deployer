/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021 Red Hat, Inc.
 */

package rte

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer"
	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/platform"
	"github.com/k8stopologyawareschedwg/deployer/pkg/deployer/wait"
	"github.com/k8stopologyawareschedwg/deployer/pkg/manifests"
	"github.com/k8stopologyawareschedwg/deployer/pkg/tlog"
)

const (
	namespaceOCP      = "openshift-monitoring"
	serviceAccountOCP = "node-exporter"
)

type Manifests struct {
	Namespace      *corev1.Namespace
	ServiceAccount *corev1.ServiceAccount
	Role           *rbacv1.Role
	RoleBinding    *rbacv1.RoleBinding
	ConfigMap      *corev1.ConfigMap
	DaemonSet      *appsv1.DaemonSet
	// internal fields
	plat           platform.Platform
	namespace      string
	serviceAccount string
}

func (mf Manifests) Clone() Manifests {
	ret := Manifests{
		plat:           mf.plat,
		namespace:      mf.namespace,
		serviceAccount: mf.serviceAccount,
		// objects
		Role:        mf.Role.DeepCopy(),
		RoleBinding: mf.RoleBinding.DeepCopy(),
		DaemonSet:   mf.DaemonSet.DeepCopy(),
	}
	if mf.plat == platform.Kubernetes {
		ret.ServiceAccount = mf.ServiceAccount.DeepCopy()
	}
	if mf.Namespace != nil {
		ret.Namespace = mf.Namespace.DeepCopy()
	}
	return ret
}

type UpdateOptions struct {
	ConfigData       string
	PullIfNotPresent bool
}

func (mf Manifests) Update(options UpdateOptions) Manifests {
	ret := mf.Clone()
	ret.ServiceAccount.Namespace = mf.namespace
	if len(options.ConfigData) > 0 {
		ret.ConfigMap = createConfigMap(mf.namespace, options.ConfigData)
	}

	ret.DaemonSet.Namespace = mf.namespace
	ret.DaemonSet.Spec.Template.Spec.ServiceAccountName = mf.serviceAccount
	ret.Role.Namespace = mf.namespace
	manifests.UpdateRoleBinding(ret.RoleBinding, mf.serviceAccount, mf.namespace)
	manifests.UpdateResourceTopologyExporterDaemonSet(ret.plat, ret.DaemonSet, ret.ConfigMap, options.PullIfNotPresent)
	return ret
}

func createConfigMap(namespace string, configData string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		// TODO: why is this needed?
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rte-config",
			Namespace: namespace,
		},
		Data: map[string]string{
			"config.yaml": configData,
		},
	}
	return cm
}

func (mf Manifests) ToObjects() []client.Object {
	objs := []client.Object{
		mf.Role,
		mf.RoleBinding,
	}
	if mf.ConfigMap != nil {
		objs = append(objs, mf.ConfigMap)
	}
	objs = append(objs, mf.DaemonSet)
	if mf.Namespace != nil {
		objs = append([]client.Object{mf.Namespace}, objs...)
	}
	if mf.plat == platform.Kubernetes {
		objs = append([]client.Object{mf.ServiceAccount}, objs...)
	}
	return objs
}

func (mf Manifests) ToCreatableObjects(hp *deployer.Helper, log tlog.Logger) []deployer.WaitableObject {
	objs := []deployer.WaitableObject{
		{Obj: mf.Role},
		{Obj: mf.RoleBinding},
		{
			Obj:  mf.DaemonSet,
			Wait: func() error { return wait.PodsToBeRunningByRegex(hp, log, mf.DaemonSet.Namespace, mf.DaemonSet.Name) },
		},
	}
	if mf.ConfigMap != nil {
		objs = append([]deployer.WaitableObject{{Obj: mf.ConfigMap}}, objs...)
	}
	if mf.plat == platform.Kubernetes {
		kubeObjs := []deployer.WaitableObject{
			{Obj: mf.Namespace},
			{Obj: mf.ServiceAccount},
		}
		return append(kubeObjs, objs...)
	}
	return objs
}

func (mf Manifests) ToDeletableObjects(hp *deployer.Helper, log tlog.Logger) []deployer.WaitableObject {
	if mf.plat == platform.Kubernetes {
		return []deployer.WaitableObject{
			{
				Obj:  mf.Namespace,
				Wait: func() error { return wait.NamespaceToBeGone(hp, log, mf.Namespace.Name) },
			},
		}
	}
	objs := []deployer.WaitableObject{
		{
			Obj:  mf.DaemonSet,
			Wait: func() error { return wait.PodsToBeGoneByRegex(hp, log, mf.DaemonSet.Namespace, mf.DaemonSet.Name) },
		},
		{Obj: mf.Role},
		{Obj: mf.RoleBinding},
	}
	if mf.ConfigMap != nil {
		objs = append(objs, deployer.WaitableObject{Obj: mf.ConfigMap})
	}
	return objs
}

func GetManifests(plat platform.Platform) (Manifests, error) {
	var err error
	mf := Manifests{
		plat: plat,
	}
	if plat == platform.Kubernetes {
		mf.Namespace, err = manifests.Namespace(manifests.ComponentResourceTopologyExporter)
		if err != nil {
			return mf, err
		}
		mf.namespace = mf.Namespace.Name

		mf.ServiceAccount, err = manifests.ServiceAccount(manifests.ComponentResourceTopologyExporter, "")
		if err != nil {
			return mf, err
		}
		mf.serviceAccount = mf.ServiceAccount.Name
	} else {
		mf.namespace = namespaceOCP
		mf.serviceAccount = serviceAccountOCP
	}
	mf.Role, err = manifests.Role(manifests.ComponentResourceTopologyExporter, "")
	if err != nil {
		return mf, err
	}
	mf.RoleBinding, err = manifests.RoleBinding(manifests.ComponentResourceTopologyExporter, "")
	if err != nil {
		return mf, err
	}
	mf.DaemonSet, err = manifests.DaemonSet(manifests.ComponentResourceTopologyExporter)
	if err != nil {
		return mf, err
	}
	return mf, nil
}

func GetManifestsForNamespace(plat platform.Platform, namespace string) (Manifests, error) {
	var err error
	mf := Manifests{
		namespace: namespace,
		plat:      plat,
	}
	if plat == platform.Kubernetes {
		mf.ServiceAccount, err = manifests.ServiceAccount(manifests.ComponentResourceTopologyExporter, "")
		if err != nil {
			return mf, err
		}
		mf.serviceAccount = mf.ServiceAccount.Name
	} else {
		mf.serviceAccount = serviceAccountOCP
	}
	mf.Role, err = manifests.Role(manifests.ComponentResourceTopologyExporter, "")
	if err != nil {
		return mf, err
	}
	mf.RoleBinding, err = manifests.RoleBinding(manifests.ComponentResourceTopologyExporter, "")
	if err != nil {
		return mf, err
	}
	mf.DaemonSet, err = manifests.DaemonSet(manifests.ComponentResourceTopologyExporter)
	if err != nil {
		return mf, err
	}
	return mf, nil
}
