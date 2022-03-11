module github.com/platform9/cluster-api-provider-cox

go 1.16

require (
	github.com/erwinvaneyk/cobras v0.0.0-20200914200705-1d2dfabe2493
	github.com/go-logr/logr v0.4.0
	github.com/olekukonko/tablewriter v0.0.4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.2.1
	go.uber.org/zap v1.19.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	k8s.io/klog/v2 v2.9.0
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b
	sigs.k8s.io/cluster-api v1.0.0
	sigs.k8s.io/controller-runtime v0.10.2
)
