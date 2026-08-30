package archive

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBinaryProcessor_NonZeroSizeRegression is the regression test for the
// silent 0-byte truncation bug (Failure B): when the destination collides with
// the source path, the old code truncated the shared inode to zero bytes and
// "installed" a 0-byte binary with no error. BinaryProcessor must never do that.
func TestBinaryProcessor_NonZeroSizeRegression(t *testing.T) {
	const content = "#!/bin/sh\necho 'a real, non-trivial binary'\n"

	t.Run("source and dest in sibling dirs", func(t *testing.T) {
		root := t.TempDir()
		srcDir := filepath.Join(root, "src")
		destDir := filepath.Join(root, "dest")
		for _, d := range []string{srcDir, destDir} {
			if err := os.MkdirAll(d, 0755); err != nil {
				t.Fatal(err)
			}
		}
		src := filepath.Join(srcDir, "mycli")
		if err := os.WriteFile(src, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}

		p := &BinaryProcessor{}
		dest, err := p.Process(src, destDir)
		if err != nil {
			t.Fatalf("BinaryProcessor.Process() error: %v", err)
		}

		assertCopiedBinary(t, src, dest, content)
	})

	t.Run("source and dest collide (old self-truncate case)", func(t *testing.T) {
		root := t.TempDir()
		src := filepath.Join(root, "mycli")
		if err := os.WriteFile(src, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}

		// dest == root and src == root/mycli means dest dir contains src; the
		// computed dest equals src exactly, replicating the historical collision.
		p := &BinaryProcessor{}
		_, err := p.Process(src, root)
		if err == nil {
			t.Fatal("expected error when destination collides with source")
		}
		if !errors.Is(err, ErrUnsupportedFormat) {
			t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
		}

		// The source must be untouched — still non-zero with original content.
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if len(b) == 0 {
			t.Fatal("source was truncated to zero bytes")
		}
		if string(b) != content {
			t.Fatalf("source content corrupted: got %q", string(b))
		}
	})
}

// assertCopiedBinary verifies the copy at dest is non-zero, matches src content,
// and that src was not truncated.
func assertCopiedBinary(t *testing.T, src, dest, content string) {
	t.Helper()

	if dest == "" {
		t.Fatal("Process() returned empty destination path")
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatalf("destination file not created: %v", err)
	}

	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("installed binary is 0 bytes")
	}
	if info.Size() != int64(len(content)) {
		t.Fatalf("installed binary size = %d, want %d", info.Size(), len(content))
	}

	b, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != content {
		t.Fatalf("installed binary content mismatch: got %q", string(b))
	}

	// The source must retain its original non-zero size (not self-truncated).
	srcInfo, err := os.Stat(src)
	if err != nil {
		t.Fatal(err)
	}
	if srcInfo.Size() == 0 {
		t.Fatal("source file was truncated to zero bytes")
	}
	if srcInfo.Size() != int64(len(content)) {
		t.Fatalf("source file size = %d, want %d", srcInfo.Size(), len(content))
	}
}

func TestArchiveProcessor_Gzip(t *testing.T) {
	src := createTestTarGz(t, "testbin", "#!/bin/sh\necho test")
	defer os.Remove(src)

	p := &ArchiveProcessor{}
	dest := t.TempDir()
	path, err := p.Process(src, dest)
	if err != nil {
		t.Fatalf("ArchiveProcessor.Process(gzip) error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("extracted file not created: %v", err)
	}
}

func TestArchiveProcessor_Zip(t *testing.T) {
	src := createTestZip(t, "testbin.exe", "test content")
	defer os.Remove(src)

	p := &ArchiveProcessor{}
	dest := t.TempDir()
	path, err := p.Process(src, dest)
	if err != nil {
		t.Fatalf("ArchiveProcessor.Process(zip) error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("extracted file not created: %v", err)
	}
}

// TestArchiveProcessor_NoExecutable verifies the ErrNoExecutable sentinel is
// returned (and matched via errors.Is) when an archive holds no executable member.
func TestArchiveProcessor_NoExecutable(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "noexec.tar.gz")
	writeTarGz(t, archivePath, "readme.txt", "hello", 0644)

	p := &ArchiveProcessor{}
	_, err := p.Process(archivePath, dir)
	if err == nil {
		t.Fatal("expected error for archive with no executable member")
	}
	if !errors.Is(err, ErrNoExecutable) {
		t.Fatalf("expected ErrNoExecutable, got %v", err)
	}
	if !strings.Contains(err.Error(), "no executable binary found") {
		t.Errorf("error message %q missing expected text", err.Error())
	}
}

// writeTarGz writes a gzip tar archive with a single member at the given mode.
func writeTarGz(t *testing.T, path, name, content string, mode int64) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	hdr := &tar.Header{
		Name: name,
		Mode: mode,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBinaryProcessor_DestIsDirectory(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	destDir := filepath.Join(root, "dest")
	for _, d := range []string{srcDir, destDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	src := filepath.Join(srcDir, "mycli")
	if err := os.WriteFile(src, []byte("content"), 0755); err != nil {
		t.Fatal(err)
	}

	p := &BinaryProcessor{}
	dest, err := p.Process(src, destDir)
	if err != nil {
		t.Fatalf("BinaryProcessor.Process() error: %v", err)
	}
	if filepath.Base(dest) != "mycli" {
		t.Errorf("destination basename = %q, want mycli", filepath.Base(dest))
	}
}
