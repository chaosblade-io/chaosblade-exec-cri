module github.com/chaosblade-io/chaosblade-exec-cri

go 1.13

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/chaosblade-io/chaosblade-exec-os v1.7.1
	github.com/chaosblade-io/chaosblade-spec-go v1.7.1
	github.com/containerd/cgroups v1.0.2-0.20210605143700-23b51209bf7b
	github.com/containerd/containerd v1.5.6
	github.com/docker/docker v20.10.21+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/gogo/protobuf v1.3.2
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/opencontainers/runtime-spec v1.0.3-0.20220909204839-494a5a6aca78
	k8s.io/cri-api v0.27.1 // indirect
	k8s.io/kubernetes v1.27.1 // indirect
)

replace (
	k8s.io/api => k8s.io/kubernetes/staging/src/k8s.io/api v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/apiextensions-apiserver => k8s.io/kubernetes/staging/src/k8s.io/apiextensions-apiserver v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/apimachinery => k8s.io/kubernetes/staging/src/k8s.io/apimachinery v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/apiserver => k8s.io/kubernetes/staging/src/k8s.io/apiserver v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/cli-runtime => k8s.io/kubernetes/staging/src/k8s.io/cli-runtime v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/client-go => k8s.io/kubernetes/staging/src/k8s.io/client-go v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/cloud-provider => k8s.io/kubernetes/staging/src/k8s.io/cloud-provider v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/cluster-bootstrap => k8s.io/kubernetes/staging/src/k8s.io/cluster-bootstrap v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/code-generator => k8s.io/kubernetes/staging/src/k8s.io/code-generator v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/component-base => k8s.io/kubernetes/staging/src/k8s.io/component-base v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/component-helpers => k8s.io/kubernetes/staging/src/k8s.io/component-helpers v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/controller-manager => k8s.io/kubernetes/staging/src/k8s.io/controller-manager v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/cri-api => k8s.io/kubernetes/staging/src/k8s.io/cri-api v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/csi-translation-lib => k8s.io/kubernetes/staging/src/k8s.io/csi-translation-lib v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/dynamic-resource-allocation => k8s.io/kubernetes/staging/src/k8s.io/dynamic-resource-allocation v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/kms => k8s.io/kubernetes/staging/src/k8s.io/kms v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/kube-aggregator => k8s.io/kubernetes/staging/src/k8s.io/kube-aggregator v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/kube-controller-manager => k8s.io/kubernetes/staging/src/k8s.io/kube-controller-manager v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/kube-proxy => k8s.io/kubernetes/staging/src/k8s.io/kube-proxy v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/kube-scheduler => k8s.io/kubernetes/staging/src/k8s.io/kube-scheduler v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/kubectl => k8s.io/kubernetes/staging/src/k8s.io/kubectl v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/kubelet => k8s.io/kubernetes/staging/src/k8s.io/kubelet v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/kubernetes => k8s.io/kubernetes v1.27.0
	k8s.io/legacy-cloud-providers => k8s.io/kubernetes/staging/src/k8s.io/legacy-cloud-providers v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/metrics => k8s.io/kubernetes/staging/src/k8s.io/metrics v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/mount-utils => k8s.io/kubernetes/staging/src/k8s.io/mount-utils v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/pod-security-admission => k8s.io/kubernetes/staging/src/k8s.io/pod-security-admission v0.0.0-20230411170423-1b4df30b3cdf
	k8s.io/sample-apiserver => k8s.io/kubernetes/staging/src/k8s.io/sample-apiserver v0.0.0-20230411170423-1b4df30b3cdf
)
