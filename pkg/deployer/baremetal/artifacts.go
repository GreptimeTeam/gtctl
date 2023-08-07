// Copyright 2023 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package baremetal

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

	"github.com/GreptimeTeam/gtctl/pkg/deployer/baremetal/config"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
	fileutils "github.com/GreptimeTeam/gtctl/pkg/utils/file"
	semverutils "github.com/GreptimeTeam/gtctl/pkg/utils/semver"
	"github.com/google/go-github/v53/github"
)

const (
	GreptimeGitHubOrg    = "GreptimeTeam"
	GreptimeDBGithubRepo = "greptimedb"
	GreptimeBinName      = "greptime"

	EtcdGitHubOrg  = "etcd-io"
	EtcdGithubRepo = "etcd"

	GOOSDarwin = "darwin"
	GOOSLinux  = "linux"

	BreakingChangeVersion = "v0.4.0-nightly-20230802"
)

// ArtifactManager is responsible for managing the artifacts of a GreptimeDB cluster.
type ArtifactManager struct {
	// dir is the global directory that contains all the artifacts.
	dir string

	// If alwaysDownload is false, the manager will not download the artifact if it already exists.
	alwaysDownload bool

	logger logger.Logger
}

type ArtifactType string

const (
	GreptimeArtifactType ArtifactType = "greptime"
	EtcdArtifactType     ArtifactType = "etcd"
)

func (t ArtifactType) String() string {
	return string(t)
}

func NewArtifactManager(workingDir string, l logger.Logger, alwaysDownload bool) (*ArtifactManager, error) {
	dir := path.Join(workingDir, "artifacts")
	if err := fileutils.CreateDirIfNotExists(dir); err != nil {
		return nil, err
	}

	return &ArtifactManager{dir: dir, alwaysDownload: alwaysDownload, logger: l}, nil
}

// BinaryPath returns the path of the binary of the given type and version.
func (am *ArtifactManager) BinaryPath(typ ArtifactType, artifact *config.Artifact) (string, error) {
	if artifact.Local != "" {
		return artifact.Local, nil
	}

	bin := path.Join(am.dir, typ.String(), artifact.Version, "bin", typ.String())
	if _, err := os.Stat(bin); os.IsNotExist(err) {
		return "", fmt.Errorf("binary not found: %s", bin)
	}
	return bin, nil
}

// PrepareArtifact will download the artifact from the given URL and uncompressed it.
func (am *ArtifactManager) PrepareArtifact(ctx context.Context, typ ArtifactType, artifact *config.Artifact) error {
	// If you use the local artifact, we don't need to download it.
	if artifact.Local != "" {
		return nil
	}

	var (
		version = artifact.Version
		pkgDir  = path.Join(am.dir, typ.String(), version, "pkg")
		binDir  = path.Join(am.dir, typ.String(), version, "bin")
	)

	artifactFile, err := am.download(ctx, typ, version, pkgDir)
	if err != nil {
		return err
	}

	// Normalize the directory structure.
	// The directory of artifacts looks like('tree -L 5 ~/.gtctl | sed 's/\xc2\xa0/ /g'):
	// ${HOME}/.gtctl
	// └── artifacts
	//    ├── etcd
	//    │   └── v3.5.7
	//    │       ├── bin
	//    │       │   ├── etcd
	//    │       │   ├── etcdctl
	//    │       │   └── etcdutl
	//    │       └── pkg
	//    │           ├── etcd-v3.5.7-darwin-arm64
	//    │           └── etcd-v3.5.7-darwin-arm64.zip
	//    └── greptime
	//        ├── latest
	//        │   ├── bin
	//        │   │   └── greptime
	//        │   └── pkg
	//        │       └── greptime-darwin-arm64.tgz
	//        └── v0.1.2
	//            ├── bin
	//            │   └── greptime
	//            └── pkg
	//                └── greptime-darwin-arm64.tgz
	switch typ {
	case GreptimeArtifactType:
		return am.installGreptime(artifactFile, binDir, version)
	case EtcdArtifactType:
		return am.installEtcd(artifactFile, pkgDir, binDir)
	default:
		return fmt.Errorf("unsupported artifact type: %s", typ)
	}
}

func (am *ArtifactManager) installEtcd(artifactFile, pkgDir, binDir string) error {
	if err := fileutils.Uncompress(artifactFile, pkgDir); err != nil {
		return err
	}

	if err := fileutils.CreateDirIfNotExists(binDir); err != nil {
		return err
	}

	artifactFile = path.Base(artifactFile)
	// If the artifactFile is '${pkgDir}/etcd-v3.5.7-darwin-arm64.zip', it will get '${pkgDir}/etcd-v3.5.7-darwin-arm64'.
	uncompressedDir := path.Join(pkgDir, artifactFile[:len(artifactFile)-len(filepath.Ext(artifactFile))])
	uncompressedDir = strings.TrimSuffix(uncompressedDir, fileutils.TarExtension)
	binaries := []string{"etcd", "etcdctl", "etcdutl"}
	for _, binary := range binaries {
		if err := fileutils.CopyFile(path.Join(uncompressedDir, binary), path.Join(binDir, binary)); err != nil {
			return err
		}
		if err := os.Chmod(path.Join(binDir, binary), 0755); err != nil {
			return err
		}
	}
	return nil
}

func (am *ArtifactManager) installGreptime(artifactFile, binDir, version string) error {
	if err := fileutils.CreateDirIfNotExists(binDir); err != nil {
		return err
	}

	if err := fileutils.Uncompress(artifactFile, binDir); err != nil {
		return err
	}

	newVersion, err := am.isBreakingVersion(version)
	if err != nil {
		return err
	}

	// If it's the breaking version, adapt to the new directory layout.
	if newVersion {
		originalBinDir := path.Join(binDir, strings.TrimSuffix(path.Base(artifactFile), fileutils.TarGzExtension))
		if err := os.Rename(path.Join(originalBinDir, GreptimeBinName), path.Join(binDir, GreptimeBinName)); err != nil {
			return err
		}
		if err := os.Remove(originalBinDir); err != nil {
			return err
		}
	}

	if err := os.Chmod(path.Join(binDir, GreptimeBinName), 0755); err != nil {
		return err
	}

	return nil
}

func (am *ArtifactManager) download(ctx context.Context, typ ArtifactType, version, pkgDir string) (string, error) {
	downloadURL, err := am.artifactURL(typ, version)
	if err != nil {
		return "", err
	}

	if err := fileutils.CreateDirIfNotExists(pkgDir); err != nil {
		return "", err
	}

	artifactFile := path.Join(pkgDir, path.Base(downloadURL))
	if !am.alwaysDownload {
		// The artifact file already exists, skip downloading.
		if _, err := os.Stat(artifactFile); err == nil {
			am.logger.V(3).Infof("The artifact file '%s' already exists, skip downloading.", artifactFile)
			return artifactFile, nil
		}

		// Other error happened, return it.
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
	}

	httpClient := &http.Client{}

	am.logger.V(3).Infof("Downloading artifact from '%s' to '%s'", downloadURL, artifactFile)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed, status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	file, err := os.Create(artifactFile)
	if err != nil {
		return "", err
	}

	_, err = file.Write(data)
	if err != nil {
		return "", err
	}

	return artifactFile, nil
}

func (am *ArtifactManager) artifactURL(typ ArtifactType, version string) (string, error) {
	switch typ {
	case GreptimeArtifactType:
		return am.greptimeDownloadURL(version)
	case EtcdArtifactType:
		return am.etcdDownloadURL(version)
	default:
		return "", fmt.Errorf("unsupported artifact type: %v", typ)
	}
}

func (am *ArtifactManager) getGreptimeLatestVersion() (string, error) {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), GreptimeGitHubOrg, GreptimeDBGithubRepo)
	if err != nil {
		return "", err
	}
	return *release.TagName, nil
}

func (am *ArtifactManager) greptimeDownloadURL(version string) (string, error) {
	if version == "latest" {
		// Get the latest greptime released version.
		latestVersion, err := am.getGreptimeLatestVersion()
		if err != nil {
			return "", err
		}
		version = latestVersion
	}

	newVersion, err := am.isBreakingVersion(version)
	if err != nil {
		return "", err
	}

	// If version >= BreakingChangeVersion, use the new download URL.
	if newVersion {
		return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s-%s-%s-%s.tar.gz",
			GreptimeGitHubOrg, GreptimeDBGithubRepo, version, string(GreptimeArtifactType), runtime.GOOS, runtime.GOARCH, version), nil
	}

	// If version < BreakingChangeVersion, use the old download URL.
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s-%s-%s.tgz",
		GreptimeGitHubOrg, GreptimeDBGithubRepo, version, string(GreptimeArtifactType), runtime.GOOS, runtime.GOARCH), nil
}

func (am *ArtifactManager) etcdDownloadURL(version string) (string, error) {
	var ext string

	switch runtime.GOOS {
	case GOOSDarwin:
		ext = fileutils.ZipExtension
	case GOOSLinux:
		ext = fileutils.TarGzExtension
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	// For the function stability, we use the specific version of etcd.
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s-%s-%s-%s%s",
		EtcdGitHubOrg, EtcdGithubRepo, version, string(EtcdArtifactType), version, runtime.GOOS, runtime.GOARCH, ext)

	return downloadURL, nil
}

func (am *ArtifactManager) isBreakingVersion(version string) (bool, error) {
	if version == "latest" {
		// Get the latest greptime released version.
		latestVersion, err := am.getGreptimeLatestVersion()
		if err != nil {
			return false, err
		}
		version = latestVersion
	}

	newVersion, err := semverutils.Compare(version, BreakingChangeVersion)
	if err != nil {
		return false, err
	}

	return newVersion || version == BreakingChangeVersion, nil
}
