package apply

import (
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"

	rukpakv1alpha1 "github.com/operator-framework/rukpak/api/v1alpha1"
)

type Apply interface {
	Apply(bd rukpakv1alpha1.BundleDeployment, chart *chart.Chart, values chartutil.Values)
}
