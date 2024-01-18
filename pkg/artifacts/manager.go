/*
 * Copyright 2023 Greptime Team
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package artifacts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/go-github/v53/github"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
	semverutils "github.com/GreptimeTeam/gtctl/pkg/utils/semver"
)

// Manager is the interface for managing artifacts.
// For now, the artifacts can be helm charts and binaries.
type Manager interface {
	// NewSource creates an artifact source with name, version, type and fromCNRegion.
	NewSource(name, version string, typ ArtifactType, fromCNRegion bool) (*Source, error)

	// DownloadTo downloads the artifact from the source to the dest and returns the path of the artifact.
	DownloadTo(ctx context.Context, from *Source, destDir string, opts *DownloadOptions) (string, error)
}

// ArtifactType is the type of the artifact.
type ArtifactType string

const (
	// ArtifactTypeChart indicates the artifact is a helm chart.
	ArtifactTypeChart ArtifactType = "chart"

	// ArtifactTypeBinary indicates the artifact is a binary.
	ArtifactTypeBinary ArtifactType = "binary"
)

// Source is the source of the artifact.
type Source struct {
	// The Name of the artifact.
	Name string

	// The FileName of the artifact.
	FileName string

	// The URL of the artifact. It can be the normal http/https URL or the OCI URL.
	URL string

	// The Version of the artifact.
	Version string

	// The type of the artifact.
	Type ArtifactType

	// Indicates whether the artifact is from the CN region.
	FromCNRegion bool
}

// DownloadOptions is the options for downloading the artifact.
type DownloadOptions struct {
	// If EnableCache is true, the manager will use the cache if the artifact already exists.
	EnableCache bool

	// If the artifact is a binary, the manager will install the binary to the BinaryInstallDir after downloading its package.
	BinaryInstallDir string
}

// manager is the implementation of Manager interface.
type manager struct {
	logger logger.Logger
}

var _ Manager = &manager{}

type Option func(*manager)

// NewManager creates a new Manager with workingDir, logger and other options.
func NewManager(logger logger.Logger, opts ...Option) (Manager, error) {
	m := &manager{
		logger: logger,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

func (m *manager) NewSource(name, version string, typ ArtifactType, fromCNRegion bool) (*Source, error) {
	src := &Source{
		Name:         name,
		Type:         typ,
		Version:      version,
		FromCNRegion: fromCNRegion,
	}

	if version == LatestVersionTag || len(version) == 0 {
		latestVersion, err := m.resolveLatestVersion(typ, name, fromCNRegion)
		if err != nil {
			return nil, err
		}
		src.Version = latestVersion
	}

	if src.Type == ArtifactTypeChart {
		src.FileName = m.chartFileName(src.Name, src.Version)
		if src.FromCNRegion {
			// The download URL example: 'https://downloads.greptime.cn/releases/charts/etcd/9.2.0/etcd-9.2.0.tgz'.
			src.URL = fmt.Sprintf("%s/%s/%s/%s", GreptimeCNCharts, src.Name, src.Version, src.FileName)
		} else {
			// Specify the OCI registry URL for the etcd chart.
			if src.Name == EtcdChartName {
				// The download URL example: 'oci://registry-1.docker.io/bitnamicharts/etcd:9.2.0'.
				src.URL = EtcdOCIRegistry
			} else {
				// The download URL example: 'https://github.com/GreptimeTeam/helm-charts/releases/download/greptimedb-0.1.1-alpha.3/greptimedb-0.1.1-alpha.3.tgz'.
				src.URL = fmt.Sprintf("%s/%s/%s", GreptimeChartReleaseDownloadURL, strings.TrimSuffix(src.FileName, fileutils.TgzExtension), src.FileName)
			}
		}
	}

	if src.Type == ArtifactTypeBinary {
		if src.Name == EtcdBinName {
			downloadURL, err := m.etcdBinaryDownloadURL(src.Version, src.FromCNRegion)
			if err != nil {
				return nil, err
			}
			src.URL = downloadURL
			src.FileName = path.Base(src.URL)
		}

		if src.Name == GreptimeBinName {
			specificVersion := src.Version
			if specificVersion == LatestVersionTag && !src.FromCNRegion {
				// Get the latest version of the latest greptime binary.
				latestVersion, err := m.latestGitHubReleaseVersion(GreptimeGitHubOrg, GreptimeDBGithubRepo)
				if err != nil {
					return nil, err
				}
				specificVersion = latestVersion
			}

			downloadURL, err := m.greptimeBinaryDownloadURL(specificVersion, src.FromCNRegion)
			if err != nil {
				return nil, err
			}
			src.URL = downloadURL
			src.FileName = path.Base(src.URL)
		}
	}

	return src, nil
}

func (m *manager) DownloadTo(ctx context.Context, from *Source, destDir string, opts *DownloadOptions) (string, error) {
	artifactFile := filepath.Join(destDir, from.FileName)
	shouldDownload := true
	if opts.EnableCache {
		_, err := os.Stat(artifactFile)

		// If the file exists, skip downloading.
		if err == nil {
			m.logger.V(3).Infof("The artifact file '%s' already exists, skip downloading.", artifactFile)
			shouldDownload = false
		}

		// Other error happened, return it.
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}

	if shouldDownload {
		m.logger.V(3).Infof("Downloading artifact from '%s' to '%s'", from.URL, destDir)

		// Ensure the directories of the destDir exist.
		if err := fileutils.EnsureDir(destDir); err != nil {
			return "", err
		}

		// Download the helm chart from OCI registry.
		if registry.IsOCI(from.URL) && from.Type == ArtifactTypeChart {
			if err := m.downloadFromOCI(from.URL, from.Version, destDir); err != nil {
				return "", err
			}
			return artifactFile, nil
		}

		if err := m.downloadFromHTTP(ctx, from.URL, artifactFile); err != nil {
			return "", err
		}
	}

	if from.Type == ArtifactTypeBinary {
		if opts.BinaryInstallDir == "" {
			return "", fmt.Errorf("binary install dir is empty")
		}
		if err := m.installBinaries(artifactFile, opts.BinaryInstallDir); err != nil {
			return "", err
		}
		return filepath.Join(filepath.Dir(destDir), "bin", from.Name), nil
	}

	return artifactFile, nil
}

func (m *manager) downloadFromHTTP(ctx context.Context, httpURL string, dest string) error {
	httpClient := &http.Client{}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, httpURL, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed, status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	file, err := os.Create(dest)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (m *manager) downloadFromOCI(registryURL, version, dest string) error {
	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(false),
		registry.ClientOptEnableCache(false),
		registry.ClientOptCredentialsFile(""),
	)
	if err != nil {
		return err
	}

	cfg := new(action.Configuration)
	cfg.RegistryClient = registryClient

	// Create a pull action
	client := action.NewPullWithOpts(action.WithConfig(cfg))
	client.Settings = cli.New()
	client.Version = version
	client.DestDir = dest

	m.logger.V(3).Infof("Pulling chart '%s', version: '%s' from OCI registry", registryURL, version)

	// Execute the pull action
	if _, err := client.Run(registryURL); err != nil {
		return err
	}

	return nil
}

// chartIndexFile returns the index file of the chart. We use the index file to get the specific version of the latest chart.
func (m *manager) chartIndexFile(ctx context.Context, indexURL string) (*repo.IndexFile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, err
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	data, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, repo.ErrEmptyIndexYaml
	}

	indexFile := &repo.IndexFile{}
	if err := yaml.UnmarshalStrict(data, &indexFile); err != nil {
		return nil, err
	}

	for _, cvs := range indexFile.Entries {
		for idx := len(cvs) - 1; idx >= 0; idx-- {
			if cvs[idx] == nil {
				continue
			}
			if cvs[idx].APIVersion == "" {
				cvs[idx].APIVersion = chart.APIVersionV1
			}
			if err := cvs[idx].Validate(); err != nil {
				cvs = append(cvs[:idx], cvs[idx+1:]...)
			}
		}
	}

	indexFile.SortEntries()
	if indexFile.APIVersion == "" {
		return indexFile, repo.ErrNoAPIVersion
	}

	return indexFile, nil
}

// latestChartVersion returns the latest chart version.
func (m *manager) latestChartVersion(indexFile *repo.IndexFile, chartName string) (*repo.ChartVersion, error) {
	if versions, ok := indexFile.Entries[chartName]; ok {
		if versions.Len() > 0 {
			// The Entries are already sorted by version so the position 0 always point to the latest version.
			v := []*repo.ChartVersion(versions)
			if len(v[0].URLs) == 0 {
				return nil, fmt.Errorf("no download URLs found for %s-%s", chartName, v[0].Version)
			}
			return v[0], nil
		}
		return nil, fmt.Errorf("chart %s has empty versions", chartName)
	}

	return nil, fmt.Errorf("chart %s not found", chartName)
}

func (m *manager) chartFileName(chartName, version string) string {
	return fmt.Sprintf("%s-%s.tgz", chartName, version)
}

// latestGitHubReleaseVersion returns the latest GitHub release version. It's used to locate the latest version of the latest greptime binary.
func (m *manager) latestGitHubReleaseVersion(org, repo string) (string, error) {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), org, repo)
	if err != nil {
		return "", err
	}
	return *release.TagName, nil
}

func (m *manager) etcdBinaryDownloadURL(version string, fromCNRegion bool) (string, error) {
	var ext string

	switch runtime.GOOS {
	case "darwin":
		ext = fileutils.ZipExtension
	case "linux":
		ext = fileutils.TarGzExtension
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	var downloadURL string
	if fromCNRegion {
		downloadURL = EtcdCNBinaries
	} else {
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/releases/download", EtcdGitHubOrg, EtcdGithubRepo)
	}

	// For the function stability, we always use the specific version of etcd.
	return fmt.Sprintf("%s/%s/etcd-%s-%s-%s%s", downloadURL, version, version, runtime.GOOS, runtime.GOARCH, ext), nil
}

func (m *manager) greptimeBinaryDownloadURL(version string, fromCNRegion bool) (string, error) {
	newVersion, err := isBreakingVersion(version)
	if err != nil {
		return "", err
	}

	var packageName string
	if newVersion {
		packageName = fmt.Sprintf("greptime-%s-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH, version)
	} else {
		packageName = fmt.Sprintf("greptime-%s-%s.tgz", runtime.GOOS, runtime.GOARCH)
	}

	var downloadURL string
	if fromCNRegion {
		downloadURL = GreptimeDBCNBinaries
	} else {
		downloadURL = fmt.Sprintf("https://github.com/%s/%s/releases/download", GreptimeGitHubOrg, GreptimeDBGithubRepo)
	}

	return fmt.Sprintf("%s/%s/%s", downloadURL, version, packageName), nil
}

// installBinaries installs the binaries to the installDir.
func (m *manager) installBinaries(downloadFile, installDir string) error {
	if err := fileutils.EnsureDir(installDir); err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("/tmp", "gtctl-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := fileutils.Uncompress(downloadFile, tempDir); err != nil {
		return err
	}

	m.logger.V(3).Infof("Installing binaries '%s' to '%s'", downloadFile, installDir)

	if err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode().IsRegular() && (info.Mode()&0111 != 0) { // Move the executable file to the installDir.
			newFilePath := filepath.Join(installDir, info.Name())
			if path != newFilePath {
				if err := os.Rename(path, newFilePath); err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// resolveLatestVersion resolves the latest tag to the specific version.
func (m *manager) resolveLatestVersion(typ ArtifactType, name string, fromCNRegion bool) (string, error) {
	if fromCNRegion {
		return m.getVersionInfoFromS3(typ, name, false)
	}

	switch typ {
	case ArtifactTypeChart:
		// Use chart index file to locate the latest chart version.
		indexFile, err := m.chartIndexFile(context.TODO(), GreptimeChartIndexURL)
		if err != nil {
			return "", err
		}

		chartVersion, err := m.latestChartVersion(indexFile, name)
		if err != nil {
			return "", err
		}
		return chartVersion.Version, nil
	case ArtifactTypeBinary:
		// Get the latest version of the latest greptime binary.
		latestVersion, err := m.latestGitHubReleaseVersion(GreptimeGitHubOrg, GreptimeDBGithubRepo)
		if err != nil {
			return "", err
		}
		return latestVersion, nil
	default:
		return "", fmt.Errorf("unsupported artifact type: %s", string(typ))
	}
}

// getVersionInfoFromS3 gets the latest version info from S3.
func (m *manager) getVersionInfoFromS3(typ ArtifactType, name string, nightly bool) (string, error) {
	// Note: it uses 'greptimedb' directory to store the greptime binary.
	if name == GreptimeBinName {
		name = "greptimedb"
	}
	var latestVersionInfoURL string
	switch typ {
	case ArtifactTypeChart:
		latestVersionInfoURL = fmt.Sprintf("%s/charts/%s/latest-version.txt", GreptimeReleaseBucketCN, name)
	case ArtifactTypeBinary:
		if nightly {
			latestVersionInfoURL = fmt.Sprintf("%s/%s/latest-nightly-version.txt", GreptimeReleaseBucketCN, name)
		} else {
			latestVersionInfoURL = fmt.Sprintf("%s/%s/latest-version.txt", GreptimeReleaseBucketCN, name)
		}
	default:
		return "", fmt.Errorf("unsupported artifact type: %s", string(typ))
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, latestVersionInfoURL, nil)
	if err != nil {
		return "", err
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get latest info from '%s' failed, status code: %d", latestVersionInfoURL, resp.StatusCode)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimRight(string(data), "\n"), nil
}

// BreakingChangeVersion is the version that the download URL of the greptime binary is changed.
const BreakingChangeVersion = "v0.4.0-nightly-20230802"

// TODO(zyy17): This function is just a temporary solution. We will remove it after the download URL of the greptime binary is stable.
func isBreakingVersion(version string) (bool, error) {
	newVersion, err := semverutils.Compare(version, BreakingChangeVersion)
	if err != nil {
		return false, err
	}

	return newVersion || version == BreakingChangeVersion, nil
}
