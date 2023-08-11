package process

import (
	"context"
	"io/fs"

	rukpakv1alpha1 "github.com/operator-framework/rukpak/api/v1alpha1"
	"github.com/varshaprasad96/rukpaklib/pkg/pipeline/process"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type placeholder struct{}

var _ process.Proccess = placeholder{}

func (p placeholder) Convert(ctx context.Context, fs fs.FS, bundle *rukpakv1alpha1.Bundle) ([]client.Object, error) {
	if isRegistry() {
		// Do conversion
	}
	return nil, nil
}

func (p placeholder) GetHelmChart([]client.Object) (*chart.Chart, chartutil.Values, error) {
	return nil, nil, nil
}

// figure out a way if its a registry bundle or not.
// should be better mechanisms than this
func isRegistry() bool {
	return true
}
