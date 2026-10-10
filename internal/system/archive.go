package system

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ArchiveManager handles compression and extraction of files and directories.
type ArchiveManager struct {
	exec *Executor
}

// NewArchiveManager creates a new ArchiveManager.
func NewArchiveManager(exec *Executor) *ArchiveManager {
	return &ArchiveManager{exec: exec}
}

// isArchiveFile checks if a filename corresponds to a supported archive format.
func isArchiveFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar.gz") ||
		strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".tar.bz2") ||
		strings.HasSuffix(lower, ".tbz2") ||
		strings.HasSuffix(lower, ".tar.xz") ||
		strings.HasSuffix(lower, ".txz") ||
		strings.HasSuffix(lower, ".tar") ||
		strings.HasSuffix(lower, ".rar") ||
		strings.HasSuffix(lower, ".7z")
}

// normalizeFormat determines canonical archive format from name or format hint.
func normalizeFormat(archiveName, format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	lower := strings.ToLower(archiveName)

	if format != "" {
		switch format {
		case "zip":
			return "zip"
		case "tar.gz", "tgz", "gz":
			return "tar.gz"
		case "tar.bz2", "tbz2", "bz2":
			return "tar.bz2"
		case "tar.xz", "txz", "xz":
			return "tar.xz"
		case "tar":
			return "tar"
		}
	}

	if strings.HasSuffix(lower, ".zip") {
		return "zip"
	} else if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return "tar.gz"
	} else if strings.HasSuffix(lower, ".tar.bz2") || strings.HasSuffix(lower, ".tbz2") {
		return "tar.bz2"
	} else if strings.HasSuffix(lower, ".tar.xz") || strings.HasSuffix(lower, ".txz") {
		return "tar.xz"
	} else if strings.HasSuffix(lower, ".tar") {
		return "tar"
	}

	return "zip"
}

// CreateArchive packages specified items within baseDir into a compressed archive.
func (am *ArchiveManager) CreateArchive(ctx context.Context, baseDir string, items []string, archiveName string, format string) (string, error) {
	baseDir = filepath.Clean(baseDir)
	if baseDir == "" || baseDir == "." {
		return "", fmt.Errorf("invalid base directory")
	}

	if len(items) == 0 {
		return "", fmt.Errorf("no items specified to archive")
	}

	archiveName = strings.TrimSpace(archiveName)
	if archiveName == "" || strings.ContainsAny(archiveName, `/\`) {
		return "", fmt.Errorf("invalid archive filename: %s", archiveName)
	}

	fmtType := normalizeFormat(archiveName, format)

	// Ensure the archive filename has the right suffix
	switch fmtType {
	case "zip":
		if !strings.HasSuffix(strings.ToLower(archiveName), ".zip") {
			archiveName += ".zip"
		}
	case "tar.gz":
		if !strings.HasSuffix(strings.ToLower(archiveName), ".tar.gz") && !strings.HasSuffix(strings.ToLower(archiveName), ".tgz") {
			archiveName += ".tar.gz"
		}
	case "tar.bz2":
		if !strings.HasSuffix(strings.ToLower(archiveName), ".tar.bz2") && !strings.HasSuffix(strings.ToLower(archiveName), ".tbz2") {
			archiveName += ".tar.bz2"
		}
	case "tar.xz":
		if !strings.HasSuffix(strings.ToLower(archiveName), ".tar.xz") && !strings.HasSuffix(strings.ToLower(archiveName), ".txz") {
			archiveName += ".tar.xz"
		}
	case "tar":
		if !strings.HasSuffix(strings.ToLower(archiveName), ".tar") {
			archiveName += ".tar"
		}
	}

	targetPath := filepath.Join(baseDir, archiveName)

	// For tar.bz2 and tar.xz or when using system tar, attempt system tar execution if available
	if fmtType == "tar.bz2" || fmtType == "tar.xz" {
		tarFlag := "-cjf"
		if fmtType == "tar.xz" {
			tarFlag = "-cJf"
		}
		args := append([]string{tarFlag, targetPath, "-C", baseDir}, items...)
		if am.exec != nil {
			_, err := am.exec.Execute(ctx, "tar", args...)
			if err != nil {
				return "", fmt.Errorf("failed to create %s archive using system tar: %w", fmtType, err)
			}
			return targetPath, nil
		}
		return "", fmt.Errorf("format %s requires system tar executor", fmtType)
	}

	// For zip, create using pure Go standard library
	if fmtType == "zip" {
		if err := am.createZip(baseDir, items, targetPath); err != nil {
			// clean up incomplete file if exists
			_ = os.Remove(targetPath)
			return "", err
		}
		return targetPath, nil
	}

	// For tar.gz and tar, create using pure Go standard library
	if fmtType == "tar.gz" || fmtType == "tar" {
		isGz := (fmtType == "tar.gz")
		if err := am.createTar(baseDir, items, targetPath, isGz); err != nil {
			_ = os.Remove(targetPath)
			return "", err
		}
		return targetPath, nil
	}

	return "", fmt.Errorf("unsupported archive format: %s", fmtType)
}

// createZip writes items into a zip archive.
func (am *ArchiveManager) createZip(baseDir string, items []string, targetPath string) error {
	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	defer zw.Close()

	for _, item := range items {
		itemClean := filepath.Clean(item)
		if strings.Contains(itemClean, "..") {
			continue
		}
		fullPath := filepath.Join(baseDir, itemClean)
		fi, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}

		if fi.IsDir() {
			err = filepath.Walk(fullPath, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				// Skip if we hit the archive file we are actively writing
				if filepath.Clean(path) == filepath.Clean(targetPath) {
					return nil
				}

				relPath, err := filepath.Rel(baseDir, path)
				if err != nil {
					return err
				}
				relSlash := filepath.ToSlash(relPath)

				if info.IsDir() {
					if !strings.HasSuffix(relSlash, "/") {
						relSlash += "/"
					}
					header := &zip.FileHeader{
						Name:     relSlash,
						Method:   zip.Store,
						Modified: info.ModTime(),
					}
					header.SetMode(info.Mode())
					_, err := zw.CreateHeader(header)
					return err
				}

				return addFileToZip(zw, path, relSlash, info)
			})
			if err != nil {
				return err
			}
		} else {
			if filepath.Clean(fullPath) == filepath.Clean(targetPath) {
				continue
			}
			relSlash := filepath.ToSlash(itemClean)
			if err := addFileToZip(zw, fullPath, relSlash, fi); err != nil {
				return err
			}
		}
	}

	return nil
}

func addFileToZip(zw *zip.Writer, fullPath, relSlash string, info os.FileInfo) error {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = relSlash
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	// If symlink, write target as content
	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(fullPath)
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(linkTarget))
		return err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	return err
}

// createTar writes items into a tar or tar.gz archive using pure Go.
func (am *ArchiveManager) createTar(baseDir string, items []string, targetPath string, isGz bool) error {
	out, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer out.Close()

	var tw *tar.Writer
	var gw *gzip.Writer

	if isGz {
		gw = gzip.NewWriter(out)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(out)
	}
	defer tw.Close()

	for _, item := range items {
		itemClean := filepath.Clean(item)
		if strings.Contains(itemClean, "..") {
			continue
		}
		fullPath := filepath.Join(baseDir, itemClean)
		fi, err := os.Lstat(fullPath)
		if err != nil {
			continue
		}

		if fi.IsDir() {
			err = filepath.Walk(fullPath, func(path string, info os.FileInfo, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if filepath.Clean(path) == filepath.Clean(targetPath) {
					return nil
				}
				relPath, err := filepath.Rel(baseDir, path)
				if err != nil {
					return err
				}
				return addFileToTar(tw, path, filepath.ToSlash(relPath), info)
			})
			if err != nil {
				return err
			}
		} else {
			if filepath.Clean(fullPath) == filepath.Clean(targetPath) {
				continue
			}
			if err := addFileToTar(tw, fullPath, filepath.ToSlash(itemClean), fi); err != nil {
				return err
			}
		}
	}

	return nil
}

func addFileToTar(tw *tar.Writer, fullPath, relSlash string, info os.FileInfo) error {
	var linkTarget string
	if info.Mode()&os.ModeSymlink != 0 {
		var err error
		linkTarget, err = os.Readlink(fullPath)
		if err != nil {
			return err
		}
	}

	header, err := tar.FileInfoHeader(info, linkTarget)
	if err != nil {
		return err
	}
	header.Name = relSlash

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	if info.Mode().IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(tw, file)
	return err
}

// ExtractArchive extracts an archive file into destDir.
func (am *ArchiveManager) ExtractArchive(ctx context.Context, archivePath, destDir string) error {
	archivePath = filepath.Clean(archivePath)
	destDir = filepath.Clean(destDir)

	if destDir == "" {
		destDir = filepath.Dir(archivePath)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	lower := strings.ToLower(archivePath)

	// Try pure Go for zip
	if strings.HasSuffix(lower, ".zip") {
		err := am.extractZip(archivePath, destDir)
		if err == nil {
			return nil
		}
		// If zip extraction failed (e.g. specialized zip format), fall through to system tar
		if am.exec != nil {
			_, sysErr := am.exec.Execute(ctx, "tar", "-xf", archivePath, "-C", destDir)
			if sysErr == nil {
				return nil
			}
		}
		return err
	}

	// Try pure Go for tar.gz and tar
	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") || strings.HasSuffix(lower, ".tar") {
		isGz := !strings.HasSuffix(lower, ".tar")
		err := am.extractTar(archivePath, destDir, isGz)
		if err == nil {
			return nil
		}
		// Fallback to system tar
		if am.exec != nil {
			_, sysErr := am.exec.Execute(ctx, "tar", "-xf", archivePath, "-C", destDir)
			if sysErr == nil {
				return nil
			}
		}
		return err
	}

	// For tar.bz2, tar.xz, 7z, rar, etc., utilize FreeBSD's libarchive bsdtar
	if am.exec != nil {
		_, err := am.exec.Execute(ctx, "tar", "-xf", archivePath, "-C", destDir)
		if err != nil {
			return fmt.Errorf("failed to extract archive with system tar: %w", err)
		}
		return nil
	}

	return fmt.Errorf("unsupported or unrecognized archive type: %s", filepath.Base(archivePath))
}

// extractZip unpacks a zip archive safely with Zip Slip protection.
func (am *ArchiveManager) extractZip(archivePath, destDir string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer zr.Close()

	destClean := filepath.Clean(destDir) + string(filepath.Separator)

	for _, f := range zr.File {
		targetPath := filepath.Join(destDir, f.Name)
		cleanTarget := filepath.Clean(targetPath)

		// Security Check: Zip Slip Prevention
		if !strings.HasPrefix(cleanTarget, destClean) && cleanTarget != filepath.Clean(destDir) {
			return fmt.Errorf("illegal zip entry path escaping destination: %s", f.Name)
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

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode()&0777)
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

// extractTar unpacks a tar or tar.gz archive safely with Tar Slip protection.
func (am *ArchiveManager) extractTar(archivePath, destDir string, isGz bool) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open tar file: %w", err)
	}
	defer f.Close()

	var tr *tar.Reader
	if isGz {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("failed to initialize gzip reader: %w", err)
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(f)
	}

	destClean := filepath.Clean(destDir) + string(filepath.Separator)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		targetPath := filepath.Join(destDir, header.Name)
		cleanTarget := filepath.Clean(targetPath)

		// Security Check: Tar Slip Prevention
		if !strings.HasPrefix(cleanTarget, destClean) && cleanTarget != filepath.Clean(destDir) {
			return fmt.Errorf("illegal tar entry path escaping destination: %s", header.Name)
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

			outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, header.FileInfo().Mode()&0777)
			if err != nil {
				return err
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()

		case tar.TypeSymlink:
			_ = os.Remove(targetPath)
			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				// symlink creation error non-fatal or ignored if not supported
				continue
			}
		}
	}

	return nil
}
