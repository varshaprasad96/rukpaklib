package fetch

import (
	"context"
	"io/fs"

	rukpakv1alpha1 "github.com/operator-framework/rukpak/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Contains result after unpacking
type Result struct {
	// Bundle contains the full filesystem of a bundle's root directory.
	Bundle fs.FS

	// ResolvedSource is a reproducible view of a Bundle's Source.
	// When possible, source implementations should return a ResolvedSource
	// that pins the Source such that future fetches of the bundle content can
	// be guaranteed to fetch the exact same bundle content as the original
	// unpack.
	//
	// For example, resolved image sources should reference a container image
	// digest rather than an image tag, and git sources should reference a
	// commit hash rather than a branch or tag.
	ResolvedSource *rukpakv1alpha1.BundleSource
}

// 1. Fetches the input from any source.
// 2. Validates input w.r.t to the required type of source
// 3. Unpacks the input.
// 4. Resturns a Result that contains bundle.
type Fetch interface {
	Validate(bundle *rukpakv1alpha1.Bundle) error
	Unpack(ctx context.Context, bundle *rukpakv1alpha1.Bundle) (*Result, error)
}

type FetchTest[source any] interface {
	Validate(source) error
	Unpack(ctx context.Context, src source) (*Result, error)
}

type git struct {
	client.Reader
	SecretNamespace string
}

type gitBundle struct {
	bundle rukpakv1alpha1.Bundle
}

var _ FetchTest[git] = &gitBundle{}

func (*gitBundle) Validate(g git) error {
	return nil
}

func (*gitBundle) Unpack(ctx context.Context, g git) (*Result, error) {
	return nil, nil
}

type imageBundle struct {
	bundle rukpakv1alpha1.Bundle
}
