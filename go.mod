module github.com/ccremer/clustercode

go 1.15

require (
	github.com/elastic/crd-ref-docs v0.0.7
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.2.0
	github.com/knadh/koanf v0.15.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.17.0
	k8s.io/api v0.20.5
	k8s.io/apimachinery v0.20.5
	k8s.io/client-go v0.20.5
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.5.0
	sigs.k8s.io/kustomize/kustomize/v3 v3.8.8
)
