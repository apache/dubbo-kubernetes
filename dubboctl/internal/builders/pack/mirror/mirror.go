package mirror

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

import (
	"github.com/buildpacks/imgutil"

	"github.com/buildpacks/pack/pkg/buildpack"
	packImage "github.com/buildpacks/pack/pkg/image"
	"github.com/buildpacks/pack/pkg/logging"

	"github.com/distribution/reference"

	"github.com/docker/docker/client"
)

type queryResult struct {
	Source    string
	Mirror    string
	Size      string
	CreatedAt string
}

type ImageReplacer func(image string) string

type MirrorFetcher struct {
	logger     logging.Logger
	fetcher    buildpack.ImageFetcher
	queryCache map[string]*queryResult
	replacer   ImageReplacer
}

var _ buildpack.ImageFetcher = &MirrorFetcher{}

var (
	supportedRegistries = map[string]struct{}{
		"gcr.io":                     {},
		"ghcr.io":                    {},
		"quay.io":                    {},
		"k8s.gcr.io":                 {},
		"docker.io":                  {},
		"registry.k8s.io":            {},
		"docker.elastic.co":          {},
		"skywalking.docker.scarf.sh": {},
	}
)

const mirrorHost = "https://docker.aityp.com"

func NewMirrorFetcher(logger logging.Logger, dockerClient client.CommonAPIClient, replacer ImageReplacer) *MirrorFetcher {
	return &MirrorFetcher{
		logger:     logger,
		fetcher:    packImage.NewFetcher(logger, dockerClient),
		queryCache: make(map[string]*queryResult),
		replacer:   replacer,
	}
}

func tagIsLatest(image reference.Named) bool {
	if t, ok := image.(reference.Tagged); ok {
		return t.Tag() == "latest"
	}
	return false
}

func (m *MirrorFetcher) queryMirror(ctx context.Context, image reference.Named) (*queryResult, error) {
	// check if the tag is latest
	imgStr := image.String()
	if tagIsLatest(image) {
		return nil, ErrLatestTagNotSupported(imgStr)
	}
	if _, isDigest := image.(reference.Digested); isDigest {
		return nil, ErrDigestNotSupported(imgStr)
	}
	if result, ok := m.queryCache[imgStr]; ok {
		if result == nil {
			return nil, ImageNotFoundError(imgStr)
		}
		return result, nil
	}
	m.logger.Debugf("pack mirror: searching for %q in mirror\n", imgStr)
	// try to query the mirror
	// see: https://docker.aityp.com/manage/api
	q := url.Values{}
	q.Add("search", imgStr)
	u := fmt.Sprintf("%s/api/v1/image?%s", mirrorHost, q.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	result := struct {
		Results []queryResult
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	var matched *queryResult
	for _, r := range result.Results {
		if r.Source == image.String() {
			matched = &r
			break
		}
	}
	m.queryCache[imgStr] = matched
	if matched == nil {
		return nil, ImageNotFoundError(imgStr)
	}
	return matched, nil
}

func (m *MirrorFetcher) replaceWithMirror(ctx context.Context, image string) (string, error) {
	if m.replacer != nil {
		m.logger.Debugf("pack mirror: replace image %q using custom replacer\n", image)
		image = m.replacer(image)
	}
	ref, err := reference.ParseDockerRef(image)
	if err != nil {
		return "", err
	}

	if _, ok := supportedRegistries[reference.Domain(ref)]; !ok {
		m.logger.Infof("pack mirror: %q is not in supported registries, skipping replacement\n", image)
		return image, nil
	}

	query, err := m.queryMirror(ctx, ref)

	switch err.(type) {
	case ImageNotFoundError:
		m.logger.Warnf("pack mirror: image not found in mirror, skipping replacement: %w\n", image, err)
		return image, nil
	case ErrLatestTagNotSupported:
		m.logger.Warnf("pack mirror: image with latest tag is not supported, skipping replacement: %w\n", image, err)
		return image, nil
	case ErrDigestNotSupported:
		m.logger.Warnf("pack mirror: image with digest is not supported, skipping replacement: %w\n", image, err)
		return image, nil
	case nil:
		m.logger.Infof("pack mirror: replaced %q with mirror %q\n", image, query.Mirror)
		return query.Mirror, nil
	default:
		return "", err
	}
}

func (m *MirrorFetcher) Fetch(ctx context.Context, name string, options packImage.FetchOptions) (imgutil.Image, error) {
	m.logger.Infof("pack mirror: fetching %q\n", name)
	name, err := m.replaceWithMirror(ctx, name)
	if err != nil {
		return nil, err
	}
	return m.fetcher.Fetch(ctx, name, options)
}

func (m *MirrorFetcher) CheckReadAccess(repo string, options packImage.FetchOptions) bool {
	m.logger.Infof("pack mirror: checking read access for %q\n", repo)
	repo, err := m.replaceWithMirror(context.Background(), repo)
	if err != nil {
		m.logger.Errorf("pack mirror: failed to check read access for %q: %v\n", repo, err)
		return false
	}
	return m.fetcher.CheckReadAccess(repo, options)
}
