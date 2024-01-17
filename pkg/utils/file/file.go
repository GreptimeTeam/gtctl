// Copyright 2023 Greptime Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

// EnsureDir ensures the directory exists.
func EnsureDir(dir string) error {
	// Check if the directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Create the directory along with any necessary parents.
		return os.MkdirAll(dir, 0755)
	}

	return nil
}

func DeleteDirIfExists(dir string) (err error) {
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func IsFileExists(filepath string) (bool, error) {
	info, err := os.Stat(filepath)
	if os.IsNotExist(err) {
		// file does not exist
		return false, nil
	}

	if err != nil {
		// Other errors happened.
		return false, err
	}

	if info.IsDir() {
		// It's a directory.
		return false, fmt.Errorf("'%s' is directory, not file", filepath)
	}

	// The file exists.
	return true, nil
}

// CopyFile copies the file from src to dst.
func CopyFile(src, dst string) error {
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

const (
	ZipExtension   = ".zip"
	TarGzExtension = ".tar.gz"
	TgzExtension   = ".tgz"
	GzExtension    = ".gz"
	TarExtension   = ".tar"
)

// Uncompress uncompresses the file to the destination directory.
func Uncompress(file, dst string) error {
	fileType := path.Ext(file)
	switch fileType {
	case ZipExtension:
		return unzip(file, dst)
	case TgzExtension, GzExtension, TarGzExtension:
		return untar(file, dst)
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}
}

func unzip(file, dst string) error {
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

		if err := dstFile.Close(); err != nil {
			return err
		}

		if err := fileInArchive.Close(); err != nil {
			return err
		}
	}

	return nil
}

func untar(file, dst string) error {
	data, err := os.ReadFile(file)
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
			filePath := path.Join(dst, header.Name)
			outFile, err := os.Create(filePath)
			if err != nil && !os.IsExist(err) {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}

			if err := os.Chmod(filePath, os.FileMode(header.Mode)); err != nil {
				return err
			}

			if err := outFile.Close(); err != nil {
				return err
			}
		case tar.TypeDir:
			if err := os.Mkdir(path.Join(dst, header.Name), 0755); err != nil && !os.IsExist(err) {
				return err
			}
		default:
			continue
		}
	}

	return nil
}
