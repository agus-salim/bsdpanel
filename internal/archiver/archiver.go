package archiver

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Compress compresses sourcePath (file or directory) into destPath based on format ("zip" or "tar.gz").
func Compress(sourcePath, destPath, format string) error {
	switch strings.ToLower(format) {
	case "zip":
		return CompressZip(sourcePath, destPath)
	case "tar.gz", "tgz":
		return CompressTarGz(sourcePath, destPath)
	default:
		return fmt.Errorf("format kompresi tidak didukung: %s", format)
	}
}

// CompressZip creates a zip archive of sourcePath at destZipPath.
func CompressZip(sourcePath, destZipPath string) error {
	outFile, err := os.Create(destZipPath)
	if err != nil {
		return fmt.Errorf("gagal membuat berkas arsip zip: %w", err)
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	fi, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("sumber tidak ditemukan: %w", err)
	}

	if !fi.IsDir() {
		// Single file
		f, err := os.Open(sourcePath)
		if err != nil {
			return err
		}
		defer f.Close()

		zf, err := w.Create(filepath.Base(sourcePath))
		if err != nil {
			return err
		}
		_, err = io.Copy(zf, f)
		return err
	}

	// Directory
	baseDir := filepath.Dir(sourcePath)
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath

		if info.IsDir() {
			header.Name += "/"
			_, err = w.CreateHeader(header)
			return err
		}

		header.Method = zip.Deflate
		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}

// CompressTarGz creates a .tar.gz archive of sourcePath at destTarGzPath.
func CompressTarGz(sourcePath, destTarGzPath string) error {
	outFile, err := os.Create(destTarGzPath)
	if err != nil {
		return fmt.Errorf("gagal membuat berkas tar.gz: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	fi, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("sumber tidak ditemukan: %w", err)
	}

	if !fi.IsDir() {
		f, err := os.Open(sourcePath)
		if err != nil {
			return err
		}
		defer f.Close()

		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.Base(sourcePath)

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err = io.Copy(tw, f)
		return err
	}

	baseDir := filepath.Dir(sourcePath)
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = relPath

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tw, file)
		return err
	})
}

// Extract extracts an archive (zip, tar.gz, tgz, tar) into destDir.
func Extract(archivePath, destDir string) error {
	destClean := filepath.Clean(destDir)
	if err := os.MkdirAll(destClean, 0755); err != nil {
		return fmt.Errorf("gagal membuat direktori tujuan: %w", err)
	}

	lower := strings.ToLower(archivePath)
	if strings.HasSuffix(lower, ".zip") {
		return extractZip(archivePath, destClean)
	} else if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return extractTarGz(archivePath, destClean)
	} else if strings.HasSuffix(lower, ".tar") {
		return extractTar(archivePath, destClean)
	}

	return fmt.Errorf("format arsip tidak dikenali (didukung: .zip, .tar.gz, .tgz, .tar)")
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("gagal membuka berkas zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		targetPath := filepath.Join(destDir, f.Name)
		// Zip slip check
		if !strings.HasPrefix(filepath.Clean(targetPath), destDir+string(os.PathSeparator)) && filepath.Clean(targetPath) != destDir {
			return fmt.Errorf("jalur ilegal di dalam berkas zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if copyErr != nil {
			return copyErr
		}
	}

	return nil
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gagal dekompresi gzip: %w", err)
	}
	defer gzr.Close()

	return extractTarReader(gzr, destDir)
}

func extractTar(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	return extractTarReader(f, destDir)
}

func extractTarReader(r io.Reader, destDir string) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, header.Name)
		// Zip slip check
		if !strings.HasPrefix(filepath.Clean(targetPath), destDir+string(os.PathSeparator)) && filepath.Clean(targetPath) != destDir {
			return fmt.Errorf("jalur ilegal di dalam arsip tar: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}
