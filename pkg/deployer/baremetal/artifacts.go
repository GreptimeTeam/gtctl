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
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/GreptimeTeam/gtctl/pkg/logger"
	"github.com/GreptimeTeam/gtctl/pkg/utils"
)

const (
	GreptimeGitHubOrg    = "GreptimeTeam"
	GreptimeDBGithubRepo = "greptimedb"

	EtcdGitHubOrg  = "etcd-io"
	EtcdGithubRepo = "etcd"

	ZipExtension   = ".zip"
	TarGzExtension = ".tar.gz"
	TarExtension   = ".tar"
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
	if err := utils.CreateDirIfNotExists(dir); err != nil {
		return nil, err
	}

	return &ArtifactManager{dir: dir, alwaysDownload: alwaysDownload, logger: l}, nil
}

// BinaryPath returns the path of the binary of the given type and version.
func (am *ArtifactManager) BinaryPath(typ ArtifactType, artifact *Artifact) (string, error) {
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
func (am *ArtifactManager) PrepareArtifact(typ ArtifactType, artifact *Artifact) error {
	// If you use the local artifact, we don't need to download it.
	if artifact.Local != "" {
		return nil
	}

	var (
		pkgDir = path.Join(am.dir, typ.String(), artifact.Version, "pkg")
		binDir = path.Join(am.dir, typ.String(), artifact.Version, "bin")
	)

	artifactFile, err := am.download(typ, artifact.Version, pkgDir)
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
		return am.installGreptime(artifactFile, binDir)
	case EtcdArtifactType:
		return am.installEtcd(artifactFile, pkgDir, binDir)
	default:
		return fmt.Errorf("unsupported artifact type: %s", typ)
	}
}

func (am *ArtifactManager) installEtcd(artifactFile, pkgDir, binDir string) error {
	if err := am.uncompress(artifactFile, pkgDir); err != nil {
		return err
	}

	if err := utils.CreateDirIfNotExists(binDir); err != nil {
		return err
	}

	artifactFile = path.Base(artifactFile)
	// If the artifactFile is '${pkgDir}/etcd-v3.5.7-darwin-arm64.zip', it will get '${pkgDir}/etcd-v3.5.7-darwin-arm64'.
	uncompressedDir := path.Join(pkgDir, artifactFile[:len(artifactFile)-len(filepath.Ext(artifactFile))])
	uncompressedDir = strings.TrimSuffix(uncompressedDir, TarExtension)
	binaries := []string{"etcd", "etcdctl", "etcdutl"}
	for _, binary := range binaries {
		if err := am.copyFile(path.Join(uncompressedDir, binary), path.Join(binDir, binary)); err != nil {
			return err
		}
		if err := os.Chmod(path.Join(binDir, binary), 0755); err != nil {
			return err
		}
	}
	return nil
}

func (am *ArtifactManager) installGreptime(artifactFile, binDir string) error {
	if err := utils.CreateDirIfNotExists(binDir); err != nil {
		return err
	}

	if err := am.uncompress(artifactFile, binDir); err != nil {
		return err
	}

	if err := os.Chmod(path.Join(binDir, "greptime"), 0755); err != nil {
		return err
	}

	return nil
}

func (am *ArtifactManager) download(typ ArtifactType, version, pkgDir string) (string, error) {
	downloadURL, err := am.artifactURL(typ, version, ZipExtension)
	if err != nil {
		return "", err
	}

	if err := utils.CreateDirIfNotExists(pkgDir); err != nil {
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

	resp, err := am.startDownload(downloadURL, httpClient)
	if err != nil {
		downloadURL, err = am.artifactURL(typ, version, TarGzExtension)
		if err != nil {
			return "", err
		}
		artifactFile = path.Join(pkgDir, path.Base(downloadURL))
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
		resp, err = am.startDownload(downloadURL, httpClient)
		if err != nil {
			return "", err
		}

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

func (am *ArtifactManager) startDownload(downloadURL string, client *http.Client) (*http.Response, error) {
	resp := &http.Response{}
	request, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return resp, err
	}
	resp, err = client.Do(request)
	if resp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("download failed, status code: %d", resp.StatusCode)
	}
	if err != nil {
		return resp, err
	}
	return resp, nil
}

func (am *ArtifactManager) artifactURL(typ ArtifactType, version, ext string) (string, error) {
	switch typ {
	case GreptimeArtifactType:
		var downloadURL string
		if version == "latest" {
			downloadURL = fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/%s-%s-%s.tgz",
				GreptimeGitHubOrg, GreptimeDBGithubRepo, string(typ), runtime.GOOS, runtime.GOARCH)
		} else {
			downloadURL = fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s-%s-%s.tgz",
				GreptimeGitHubOrg, GreptimeDBGithubRepo, version, string(typ), runtime.GOOS, runtime.GOARCH)
		}
		return downloadURL, nil
	case EtcdArtifactType:
		// For the function stability, we use the specific version of etcd.
		downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s-%s-%s-%s%s",
			EtcdGitHubOrg, EtcdGithubRepo, version, string(typ), version, runtime.GOOS, runtime.GOARCH, ext)
		return downloadURL, nil
	default:
		return "", fmt.Errorf("unsupported artifact type: %v", typ)
	}
}

func (am *ArtifactManager) uncompress(file, dst string) error {
	fileType := path.Ext(file)
	switch fileType {
	case ".zip":
		return am.unzip(file, dst)
	case ".tgz":
		return am.untar(file, dst)
	case ".gz":
		return am.untar(file, dst)
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}
}

func (am *ArtifactManager) unzip(file, dst string) error {
	archive, err := zip.OpenReader(file)
	if err != nil {
		return err
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			return err
		}

		dstFile.Close()
		fileInArchive.Close()
	}

	return nil
}

func (am *ArtifactManager) untar(file, dst string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	stream, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(stream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeReg:
			outFile, err := os.Create(dst + "/" + header.Name)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
			outFile.Close()
		case tar.TypeDir:
			if err := os.Mkdir(dst+"/"+header.Name, 0755); err != nil {
				return err
			}
		default:
			continue
		}
	}

	return nil
}

func (am *ArtifactManager) copyFile(src, dst string) error {
	r, err := os.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	if err != nil {
		return err
	}

	return w.Sync()
}
