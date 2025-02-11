// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

//go:build e2eingressclass
// +build e2eingressclass

package runner

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	versioned "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var _ = Describe("extensions-v1beta1-IngressClass", func() {
	var (
		clientset *kubernetes.Clientset
		crdClient *versioned.Clientset
		err       error
	)

	Context("Test Ingress Class", func() {
		BeforeEach(func() {
			clientset, crdClient, err = getClients()
			Expect(err).To(BeNil())

			UseExtensionsV1Beta1Ingress = supportsExtensionsV1Beta1IngressPackage(clientset)
			skipIfExtensionsV1Beta1NotSupport()

			cleanUp(clientset)
		})

		It("[ingress-class] should only pick up ingresses that match the class specified in helm", func() {
			namespaceName := "e2e-ingress-class"
			ns := &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
			}
			klog.Info("Creating namespace: ", namespaceName)
			_, err = clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			Expect(err).To(BeNil())

			// This application has two ingresses:
			// 1. One Ingress: uses "azure/application-gateway" as ingress class
			// 2. Two Ingress: uses "custom-ingress-class" as ingress class
			// We expect that AGIC will use ingress with "custom-ingress-class"
			SSLIngressClassYamlPath := "testdata/extensions-v1beta1/ingress-class/app.yaml"
			klog.Info("Applying yaml: ", SSLIngressClassYamlPath)
			err = applyYaml(clientset, crdClient, namespaceName, SSLIngressClassYamlPath)
			Expect(err).To(BeNil())
			time.Sleep(30 * time.Second)

			// get ip address for 1 ingress
			klog.Info("Getting public IP from Ingress...")
			publicIP, _ := getPublicIP(clientset, namespaceName)
			Expect(publicIP).ToNot(Equal(""))

			url := fmt.Sprintf("http://%s/status/200", publicIP)

			// should return 404 as this ingress is using "azure/application-gateway"
			_, err = makeGetRequest(url, "www.default.com", 404, true)
			Expect(err).To(BeNil(), "This should return 404 as this ingress is using 'azure/application-gateway'")

			// should return 200 as this ingress is using "custom-ingress-class"
			_, err = makeGetRequest(url, "www.custom.com", 200, true)
			Expect(err).To(BeNil(), "This should return 200 as this ingress is using 'custom-ingress-class'")
		})

		AfterEach(func() {
			// clear all namespaces
			cleanUp(clientset)
		})
	})
})
