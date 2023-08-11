package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/varshaprasad96/rukpaklib/pkg/pipeline/fetch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rukpakv1alpha1 "github.com/operator-framework/rukpak/api/v1alpha1"
)

type gitSource struct {
	client.Reader
	SecretNamespace string
}

var _ fetch.Fetch = &gitSource{}

func (g *gitSource) Validate(bundle *rukpakv1alpha1.Bundle) error {
	if bundle.Spec.Source.Type != rukpakv1alpha1.SourceTypeGit {
		return fmt.Errorf("bundle source type %q not supported", bundle.Spec.Source.Type)
	}
	if bundle.Spec.Source.Git == nil {
		return fmt.Errorf("bundle source git configuration is unset")
	}

	gitsource := bundle.Spec.Source.Git
	if gitsource.Repository == "" {
		// This should never happen because the validation webhook rejects git bundles without repository
		return errors.New("missing git source information: repository must be provided")
	}
	return nil
}

func (g *gitSource) Unpack(ctx context.Context, bundle *rukpakv1alpha1.Bundle) (*fetch.Result, error) {
	progress := bytes.Buffer{}
	gitsource := bundle.Spec.Source.Git
	cloneOpts := git.CloneOptions{
		URL:             gitsource.Repository,
		Progress:        &progress,
		Tags:            git.NoTags,
		InsecureSkipTLS: bundle.Spec.Source.Git.Auth.InsecureSkipVerify,
	}

	// if bundle.Spec.Source.Git.Auth.Secret.Name != "" {
	// 	auth, err := g.configAuth(ctx, bundle)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("configuring Auth error: %w", err)
	// 	}
	// 	cloneOpts.Auth = auth
	// }

	if gitsource.Ref.Branch != "" {
		cloneOpts.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", gitsource.Ref.Branch))
		cloneOpts.SingleBranch = true
		cloneOpts.Depth = 1
	} else if gitsource.Ref.Tag != "" {
		cloneOpts.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", gitsource.Ref.Tag))
		cloneOpts.SingleBranch = true
		cloneOpts.Depth = 1
	}

	// Clone
	repo, err := git.CloneContext(ctx, memory.NewStorage(), memfs.New(), &cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("bundle unpack git clone error: %v - %s", err, progress.String())
	}
	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("bundle unpack error: %v", err)
	}

	// Checkout commit
	if gitsource.Ref.Commit != "" {
		commitHash := plumbing.NewHash(gitsource.Ref.Commit)
		if err := wt.Reset(&git.ResetOptions{
			Commit: commitHash,
			Mode:   git.HardReset,
		}); err != nil {
			return nil, fmt.Errorf("checkout commit %q: %v", commitHash.String(), err)
		}
	}

	var bundleFS fs.FS = &billyFS{wt.Filesystem}

	// Subdirectory
	if gitsource.Directory != "" {
		directory := filepath.Clean(gitsource.Directory)
		if directory[:3] == "../" || directory[0] == '/' {
			return nil, fmt.Errorf("get subdirectory %q for repository %q: %s", gitsource.Directory, gitsource.Repository, "directory can not start with '../' or '/'")
		}
		sub, err := wt.Filesystem.Chroot(filepath.Clean(directory))
		if err != nil {
			return nil, fmt.Errorf("get subdirectory %q for repository %q: %v", gitsource.Directory, gitsource.Repository, err)
		}
		bundleFS = &billyFS{sub}
	}

	commitHash, err := repo.ResolveRevision("HEAD")
	if err != nil {
		return nil, fmt.Errorf("resolve commit hash: %v", err)
	}

	resolvedGit := bundle.Spec.Source.Git.DeepCopy()
	resolvedGit.Ref = rukpakv1alpha1.GitRef{
		Commit: commitHash.String(),
	}

	resolvedSource := &rukpakv1alpha1.BundleSource{
		Type: rukpakv1alpha1.SourceTypeGit,
		Git:  resolvedGit,
	}

	return &fetch.Result{Bundle: bundleFS, ResolvedSource: resolvedSource}, nil
}

var (
	_ fs.FS         = &billyFS{}
	_ fs.ReadDirFS  = &billyFS{}
	_ fs.ReadFileFS = &billyFS{}
	_ fs.StatFS     = &billyFS{}
	_ fs.File       = &billyFile{}
)

type billyFS struct {
	billy.Filesystem
}

func (f *billyFS) ReadFile(name string) ([]byte, error) {
	file, err := f.Filesystem.Open(name)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(file)
}

func (f *billyFS) Open(path string) (fs.File, error) {
	fi, err := f.Filesystem.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return &billyDirFile{billyFile{nil, fi}, f, path}, nil
	}
	file, err := f.Filesystem.Open(path)
	if err != nil {
		return nil, err
	}
	return &billyFile{file, fi}, nil
}

func (f *billyFS) ReadDir(name string) ([]fs.DirEntry, error) {
	fis, err := f.Filesystem.ReadDir(name)
	if err != nil {
		return nil, err
	}
	entries := make([]fs.DirEntry, 0, len(fis))
	for _, fi := range fis {
		entries = append(entries, fs.FileInfoToDirEntry(fi))
	}
	return entries, nil
}

type billyFile struct {
	billy.File
	fi os.FileInfo
}

func (b billyFile) Stat() (fs.FileInfo, error) {
	return b.fi, nil
}

func (b billyFile) Close() error {
	if b.File == nil {
		return nil
	}
	return b.File.Close()
}

type billyDirFile struct {
	billyFile
	fs   *billyFS
	path string
}

func (d *billyDirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	entries, err := d.fs.ReadDir(d.path)
	if n <= 0 || n > len(entries) {
		n = len(entries)
	}
	return entries[:n], err
}

func (d billyDirFile) Read(_ []byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.path, Err: syscall.EISDIR}
}
