package process

import (
	"context"
	"io/fs"

	rukpakv1alpha1 "github.com/operator-framework/rukpak/api/v1alpha1"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Proccess interface {
	Convert(context.Context, fs.FS, *rukpakv1alpha1.Bundle) ([]client.Object, error)
	GetHelmChart([]client.Object) (*chart.Chart, chartutil.Values, error)
}
