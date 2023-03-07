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
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	. "github.com/GreptimeTeam/gtctl/pkg/deployer"
	"github.com/GreptimeTeam/gtctl/pkg/logger"
)

type deployer struct {
	logger logger.Logger
	config *Config

	wg sync.WaitGroup
}

var _ Deployer = &deployer{}

type Option func(*deployer)

func NewDeployer(l logger.Logger, clusterName string, opts ...Option) (Deployer, error) {
	d := &deployer{
		logger: l,
		config: defaultConfig(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(d)
		}
	}

	if err := d.prepare(clusterName); err != nil {
		return nil, err
	}

	return d, nil
}

func WithConfig(config *Config) Option {
	// TODO(zyy17): Should merge the default configuration.
	return func(d *deployer) {
		d.config = config
	}
}

func (d *deployer) GetGreptimeDBCluster(ctx context.Context, name string, options *GetGreptimeDBClusterOptions) (*GreptimeDBCluster, error) {
	return nil, fmt.Errorf("unsupported operation")
}

func (d *deployer) ListGreptimeDBClusters(ctx context.Context, options *ListGreptimeDBClustersOptions) ([]*GreptimeDBCluster, error) {
	return nil, fmt.Errorf("unsupported operation")
}

func (d *deployer) CreateGreptimeDBCluster(ctx context.Context, clusterName string, options *CreateGreptimeDBClusterOptions) error {
	var (
		greptimeBinaryFile string

		workingDir = filepath.Join(GtctlDir, clusterName)
	)

	// TODO(zyy17): Add the validation of the options.
	if d.config.Cluster == nil || d.config.Cluster.Artifact == nil {
		return fmt.Errorf("invalid config")
	}

	if d.config.Cluster.Artifact.Version == "" && d.config.Cluster.Artifact.Local == "" {
		return fmt.Errorf("empty artifact")
	}

	if d.config.Cluster.Datanode == nil {
		return fmt.Errorf("invalid datanode")
	}

	if d.config.Cluster.Datanode.Replicas <= 0 {
		return fmt.Errorf("invalid replicas '%d'", d.config.Cluster.Datanode.Replicas)
	}

	// Download the greptime binary from GitHub
	if d.config.Cluster.Artifact.Version != "" {
		greptimeBinaryDownloadURL := fmt.Sprintf("%s/%s/greptime-%s-%s.tgz",
			GreptimeBinaryDownloadURLPrefix, d.config.Cluster.Artifact.Version, runtime.GOOS, runtime.GOARCH)
		greptimeCompressedFile := path.Base(greptimeBinaryDownloadURL)
		packagesDir := filepath.Join(workingDir, PackagesDir)
		binaryDir := filepath.Join(workingDir, BinaryDir)
		if err := d.downloadBinary(greptimeBinaryDownloadURL, path.Join(packagesDir, greptimeCompressedFile)); err != nil {
			return err
		}

		// Untar the greptime binary.
		if err := d.untar(path.Join(packagesDir, greptimeCompressedFile), packagesDir); err != nil {
			return err
		}

		// Move uncompressed greptime binary to bin/.
		greptimeUncompressedFile := path.Join(packagesDir, "greptime")
		if err := os.Chmod(greptimeUncompressedFile, 0755); err != nil {
			return err
		}

		greptimeBinaryDir := path.Join(binaryDir, "greptime")
		if err := d.createDirIfNotExists(greptimeBinaryDir); err != nil {
			return err
		}
		if err := os.Rename(greptimeUncompressedFile, path.Join(greptimeBinaryDir, "greptime")); err != nil {
			return err
		}
		greptimeBinaryFile = path.Join(workingDir, BinaryDir, "greptime", "greptime")
	}

	// Use the local binary.
	if d.config.Cluster.Artifact.Local != "" {
		greptimeBinaryFile = d.config.Cluster.Artifact.Local
		// check if the binary exists.
		if _, err := os.Stat(greptimeBinaryFile); err != nil {
			return err
		}
	}

	// Start metasrv.
	metasrvLogDir := path.Join(workingDir, LogsDir, "metasrv")
	if err := d.createDirIfNotExists(metasrvLogDir); err != nil {
		return err
	}

	metasrvPidDir := path.Join(workingDir, PidsDir, "metasrv")
	if err := d.createDirIfNotExists(metasrvPidDir); err != nil {
		return err
	}

	err := d.runBinary(greptimeBinaryFile, d.buildMetaStartArgs(), metasrvLogDir, metasrvPidDir)
	if err != nil {
		return err
	}

	// FIXME(zyy17): Wait for the metasrv to start. The datanode will fail to start if the metasrv is not ready.
	time.Sleep(2 * time.Second)

	for i := 0; i < d.config.Cluster.Datanode.Replicas; i++ {
		dirName := fmt.Sprintf("datanode.%d", i)
		// Start datanode.
		datanodeLogDir := path.Join(workingDir, LogsDir, dirName)
		if err := d.createDirIfNotExists(datanodeLogDir); err != nil {
			return err
		}

		datanodePidDir := path.Join(workingDir, PidsDir, dirName)
		if err := d.createDirIfNotExists(datanodePidDir); err != nil {
			return err
		}

		datanodeDataDir := path.Join(workingDir, DataDir, dirName)
		if err := d.createDirIfNotExists(datanodeDataDir); err != nil {
			return err
		}

		err := d.runBinary(greptimeBinaryFile, d.buildDatanodeArgs(i, datanodeDataDir), datanodeLogDir, datanodePidDir)
		if err != nil {
			return err
		}
	}

	// Start frontend.
	frontendLogDir := path.Join(workingDir, LogsDir, "frontend")
	if err := d.createDirIfNotExists(frontendLogDir); err != nil {
		return err
	}

	frontendPidDir := path.Join(workingDir, PidsDir, "frontend")
	if err := d.createDirIfNotExists(frontendPidDir); err != nil {
		return err
	}

	err = d.runBinary(greptimeBinaryFile, d.buildFrontendArgs(), frontendLogDir, frontendPidDir)
	if err != nil {
		return err
	}

	return nil
}

func (d *deployer) UpdateGreptimeDBCluster(ctx context.Context, name string, options *UpdateGreptimeDBClusterOptions) error {
	return fmt.Errorf("unsupported operation")
}

func (d *deployer) DeleteGreptimeDBCluster(ctx context.Context, name string, options *DeleteGreptimeDBClusterOption) error {
	return fmt.Errorf("unsupported operation")
}

func (d *deployer) CreateEtcdCluster(ctx context.Context, clusterName string, options *CreateEtcdClusterOptions) error {
	var (
		etcdBinaryFile string

		workingDir  = filepath.Join(GtctlDir, clusterName)
		etcdDataDir = path.Join(workingDir, DataDir, "etcd")
		etcdLogDir  = path.Join(workingDir, LogsDir, "etcd")
		etcdPidDir  = path.Join(workingDir, PidsDir, "etcd")
		etcdDirs    = []string{etcdDataDir, etcdLogDir, etcdPidDir}
	)

	// TODO(zyy17): Add the validation of the options.
	if d.config.Cluster == nil || d.config.Cluster.Artifact == nil {
		return fmt.Errorf("invalid config")
	}

	if d.config.Etcd.Artifact.Version == "" && d.config.Etcd.Artifact.Local == "" {
		return fmt.Errorf("empty artifact")
	}

	for _, dir := range etcdDirs {
		if err := d.createDirIfNotExists(dir); err != nil {
			return err
		}
	}

	// Download the greptime binary from GitHub
	if d.config.Etcd.Artifact.Version != "" {
		// FIXME(zyy17): detect the existence of etcd cluster.
		etcdBinaryDownloadURL := fmt.Sprintf("%s%s-%s.zip", EtcdBinaryDownloadURLPrefix, runtime.GOOS, runtime.GOARCH)
		etcdCompressedFile := path.Base(etcdBinaryDownloadURL)
		packagesDir := filepath.Join(workingDir, PackagesDir)
		binaryDir := filepath.Join(workingDir, BinaryDir)
		if err := d.downloadBinary(etcdBinaryDownloadURL, path.Join(packagesDir, etcdCompressedFile)); err != nil {
			return err
		}

		// Unzip the etcd binary.
		if err := d.unzip(path.Join(packagesDir, etcdCompressedFile), packagesDir); err != nil {
			return err
		}

		// Move uncompressed etcd binary to bin/.
		etcdUncompressedDir := path.Join(packagesDir, etcdCompressedFile[:len(etcdCompressedFile)-len(filepath.Ext(etcdCompressedFile))])
		if err := os.Rename(etcdUncompressedDir, path.Join(binaryDir, "etcd")); err != nil {
			return err
		}

		etcdBinaryFile = path.Join(workingDir, BinaryDir, "etcd", "etcd")
	}

	// Use the local binary.
	if d.config.Etcd.Artifact.Local != "" {
		etcdBinaryFile = d.config.Etcd.Artifact.Local
		// check if the binary exists.
		if _, err := os.Stat(etcdBinaryFile); err != nil {
			return err
		}
	}

	args := []string{"--data-dir", etcdDataDir}
	go func() {
		err := d.runBinary(etcdBinaryFile, args, etcdLogDir, etcdPidDir)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

func (d *deployer) DeleteEtcdCluster(ctx context.Context, name string, options *DeleteEtcdClusterOption) error {
	return fmt.Errorf("unsupported operation")
}

func (d *deployer) CreateGreptimeDBOperator(ctx context.Context, name string, options *CreateGreptimeDBOperatorOptions) error {
	// We don't need to implement this method because we don't need to deploy GreptimeDB Operator.
	return fmt.Errorf("only support for k8s deployer")
}

func (d *deployer) Wait(ctx context.Context) error {
	d.wg.Wait()
	return nil
}

func (d *deployer) prepare(clusterName string) error {
	dirs := []string{
		// ${PWD}/${GtctlDir}.
		path.Join(GtctlDir),

		// ${PWD}/${GtctlDir}/${ClusterName}.
		path.Join(GtctlDir, clusterName),

		// ${PWD}/${GtctlDir}/${ClusterName}/${PackagesDir}.
		path.Join(GtctlDir, clusterName, PackagesDir),

		// ${PWD}/${GtctlDir}/${ClusterName}/${BinaryDir}.
		path.Join(GtctlDir, clusterName, BinaryDir),

		// ${PWD}/${GtctlDir}/${ClusterName}/${LogsDir}.
		path.Join(GtctlDir, clusterName, LogsDir),

		// ${PWD}/${GtctlDir}/${ClusterName}/${PidsDir}.
		path.Join(GtctlDir, clusterName, PidsDir),

		// ${PWD}/${GtctlDir}/${ClusterName}/${DataDir}.
		path.Join(GtctlDir, clusterName, DataDir),
	}

	for _, dir := range dirs {
		if err := d.createDirIfNotExists(dir); err != nil {
			return err
		}
	}

	return nil
}

func (d *deployer) downloadBinary(url, dst string) error {
	httpClient := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	file, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (d *deployer) createDirIfNotExists(dir string) (err error) {
	// Create the directory if not exists.
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		if err = os.Mkdir(dir, 0755); err != nil {
			return err
		}
	}

	// Ignore the error if the directory already exists.
	if err != nil && os.IsExist(err) {
		return nil
	}

	return err
}

func (d *deployer) unzip(file, dst string) error {
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

func (d *deployer) untar(file, dst string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	stream, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(stream)

	for true {
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

func (d *deployer) runBinary(binary string, args []string, logDir string, pidDir string) error {
	cmd := exec.Command(binary, args...)

	// output to binary.
	logFile := path.Join(logDir, "log")
	outputFile, err := os.Create(logFile)
	if err != nil {
		return err
	}

	outputFileWriter := bufio.NewWriter(outputFile)
	cmd.Stdout = outputFileWriter
	cmd.Stderr = outputFileWriter

	if err := cmd.Start(); err != nil {
		return err
	}

	pidFile := path.Join(pidDir, "pid")
	f, err := os.Create(pidFile)
	if err != nil {
		return err
	}

	_, err = f.Write([]byte(strconv.Itoa(cmd.Process.Pid)))
	if err != nil {
		return err
	}

	go func() {
		defer d.wg.Done()
		d.wg.Add(1)
		if err := cmd.Wait(); err != nil {
			panic(err)
		}
	}()

	return nil
}

func (d *deployer) buildMetaStartArgs() []string {
	args := []string{
		"metasrv", "start",
		"--store-addr", d.config.Cluster.Meta.StoreAddr,
		"--server-addr", d.config.Cluster.Meta.ServerAddr,
	}
	return args
}

func (d *deployer) buildDatanodeArgs(nodeID int, dataDir string) []string {
	rpcPort := d.config.Cluster.Datanode.RPCPort + nodeID
	mysqlPort := d.config.Cluster.Datanode.MySQLPort + nodeID
	args := []string{
		"datanode", "start",
		fmt.Sprintf("--node-id=%d", nodeID),
		fmt.Sprintf("--metasrv-addr=%s", d.config.Cluster.Meta.ServerAddr),
		fmt.Sprintf("--rpc-addr=0.0.0.0:%d", rpcPort),
		fmt.Sprintf("--mysql-addr=0.0.0.0:%d", mysqlPort),
		fmt.Sprintf("--data-dir=%s", path.Join(dataDir, "data")),
		fmt.Sprintf("--wal-dir=%s", path.Join(dataDir, "wal")),
	}
	return args
}

func (d *deployer) buildFrontendArgs() []string {
	args := []string{
		"frontend", "start",
		fmt.Sprintf("--metasrv-addr=%s", d.config.Cluster.Meta.ServerAddr),
	}
	return args
}
