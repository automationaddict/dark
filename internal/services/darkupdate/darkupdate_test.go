package darkupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.1.0", "0.1.0", 0},
		{"v0.1.0", "0.1.0", 0},
		{"0.1.1", "0.1.0", 1},
		{"0.1.0", "0.1.1", -1},
		{"0.2.0", "0.1.9", 1},
		{"1.0.0", "0.9.9", 1},
		{"v0.1.0", "v0.1.0-dev", 1},   // release > prerelease of same base
		{"0.1.0-dev", "0.1.0", -1},
		{"0.1.0", "0.1.0-rc1", 1},
		{"0.1.0-rc1", "0.1.0-rc2", -1},
		{"0.2.0", "0.2.0-dev", 1},
	}
	for _, tt := range cases {
		if got := compareVersions(tt.a, tt.b); got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestVerifyChecksum round-trips a checksum file through the same
// sha256sum format install.sh writes, and exercises both the
// happy path and the tampered-tarball path.
func TestVerifyChecksum(t *testing.T) {
	dir := t.TempDir()

	tarballName := "dark-v0.1.0-linux-amd64.tar.gz"
	tarball := filepath.Join(dir, tarballName)
	content := []byte("pretend this is a real tarball")
	if err := os.WriteFile(tarball, content, 0o644); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(content)
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	sums := fmt.Sprintf("%s  %s\n", hex.EncodeToString(sum[:]), tarballName)
	if err := os.WriteFile(sumsPath, []byte(sums), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := verifyChecksum(tarball, sumsPath, tarballName); err != nil {
		t.Errorf("verifyChecksum on matching file: %v", err)
	}

	// Tamper with the tarball and confirm verify fails.
	if err := os.WriteFile(tarball, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksum(tarball, sumsPath, tarballName); err == nil {
		t.Error("verifyChecksum should fail on tampered file")
	}
}

func TestVerifyChecksumMissingEntry(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "present.tar.gz")
	_ = os.WriteFile(tarball, []byte("x"), 0o644)
	sumsPath := filepath.Join(dir, "SHA256SUMS")
	// SHA256SUMS doesn't mention present.tar.gz.
	_ = os.WriteFile(sumsPath, []byte("deadbeef  other.tar.gz\n"), 0o644)

	if err := verifyChecksum(tarball, sumsPath, "present.tar.gz"); err == nil {
		t.Error("verifyChecksum should fail when entry is missing")
	}
}

// TestExtractTarball builds a real gzipped tarball in memory with
// a couple of binaries and extracts it, confirming the files land
// at the expected paths with the expected contents.
func TestExtractTarball(t *testing.T) {
	dir := t.TempDir()
	tarballPath := filepath.Join(dir, "release.tar.gz")

	files := map[string]string{
		"dark":        "fake dark binary",
		"darkd":       "fake daemon binary",
		"dark-helper": "fake helper binary",
	}
	writeFakeTarball(t, tarballPath, files)

	extractDir := filepath.Join(dir, "stage")
	if err := extractTarball(tarballPath, extractDir); err != nil {
		t.Fatalf("extractTarball: %v", err)
	}

	for name, want := range files {
		p := filepath.Join(extractDir, name)
		got, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("read %s: %v", p, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s content mismatch: got %q, want %q", name, got, want)
		}
		info, err := os.Stat(p)
		if err == nil && info.Mode()&0o100 == 0 {
			t.Errorf("%s should be executable, got mode %v", name, info.Mode())
		}
	}
}

// TestExtractTarballRejectsTraversal ensures a malicious tarball
// with ../ paths can't escape the extract directory.
func TestExtractTarballRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	tarballPath := filepath.Join(dir, "evil.tar.gz")

	// Build a tarball with a traversal entry manually.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte("evil")
	hdr := &tar.Header{
		Name:     "../escape.txt",
		Mode:     0o644,
		Size:     int64(len(body)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()
	_ = os.WriteFile(tarballPath, buf.Bytes(), 0o644)

	extractDir := filepath.Join(dir, "stage")
	err := extractTarball(tarballPath, extractDir)
	if err == nil {
		t.Fatal("extractTarball should reject traversal paths")
	}
}

// TestInstallBinariesAtomic writes fake binaries into a stage dir
// and confirms installBinaries moves them to the install prefix.
// Uses HOME override so we don't touch the real ~/.local/bin.
func TestInstallBinariesAtomic(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	stage := filepath.Join(home, "stage")
	if err := os.MkdirAll(stage, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, bin := range binaryNames {
		p := filepath.Join(stage, bin)
		if err := os.WriteFile(p, []byte("bin-"+bin), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	if err := installBinaries(stage); err != nil {
		t.Fatalf("installBinaries: %v", err)
	}

	prefix := filepath.Join(home, ".local", "bin")
	for _, bin := range binaryNames {
		p := filepath.Join(prefix, bin)
		got, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("install missed %s: %v", bin, err)
			continue
		}
		if string(got) != "bin-"+bin {
			t.Errorf("%s content mismatch: %q", bin, got)
		}
		// The staging .new file shouldn't be left behind.
		tmp := p + ".new"
		if _, err := os.Stat(tmp); err == nil {
			t.Errorf("%s.new stage file should be gone", bin)
		}
	}
}

// writeFakeTarball is a test helper that writes a gzipped tar with
// the given name→content map as executable regular files.
func writeFakeTarball(t *testing.T, path string, files map[string]string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o755,
			Size:     int64(len(body)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}
