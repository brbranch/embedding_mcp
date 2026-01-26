package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCanonicalizeProjectID_TildeExpand はチルダ展開が正しく行われることをテスト
func TestCanonicalizeProjectID_TildeExpand(t *testing.T) {
	projectID := "~/test/project"
	canonical, err := CanonicalizeProjectID(projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expected := filepath.Join(home, "test", "project")
	if canonical != expected {
		t.Errorf("expected %q, got %q", expected, canonical)
	}
}

// TestCanonicalizeProjectID_AbsolutePath は相対パスが絶対パスに変換されることをテスト
func TestCanonicalizeProjectID_AbsolutePath(t *testing.T) {
	// 一時ディレクトリを作成
	tmpDir := t.TempDir()
	relPath := "test/project"
	fullPath := filepath.Join(tmpDir, relPath)

	// 作業ディレクトリを一時ディレクトリに変更
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// 相対パスを正規化
	canonical, err := CanonicalizeProjectID(relPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !filepath.IsAbs(canonical) {
		t.Errorf("expected absolute path, got %q", canonical)
	}

	if canonical != fullPath {
		t.Errorf("expected %q, got %q", fullPath, canonical)
	}
}

// TestCanonicalizeProjectID_Symlink はシンボリックリンクが解決されることをテスト
func TestCanonicalizeProjectID_Symlink(t *testing.T) {
	tmpDir := t.TempDir()

	// 実際のディレクトリとシンボリックリンクを作成
	realDir := filepath.Join(tmpDir, "real")
	linkDir := filepath.Join(tmpDir, "link")

	if err := os.Mkdir(realDir, 0755); err != nil {
		t.Fatalf("failed to create real dir: %v", err)
	}

	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// シンボリックリンクを正規化
	canonical, err := CanonicalizeProjectID(linkDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if canonical != realDir {
		t.Errorf("expected %q, got %q", realDir, canonical)
	}
}

// TestCanonicalizeProjectID_SymlinkFail はシンボリックリンク解決失敗時に絶対パスまで処理することをテスト
func TestCanonicalizeProjectID_SymlinkFail(t *testing.T) {
	tmpDir := t.TempDir()

	// 存在しないパスへのシンボリックリンクを作成
	linkDir := filepath.Join(tmpDir, "broken-link")
	if err := os.Symlink("/nonexistent/path", linkDir); err != nil {
		t.Fatalf("failed to create broken symlink: %v", err)
	}

	// シンボリックリンク解決は失敗するが、絶対パスまでは取得できる
	canonical, err := CanonicalizeProjectID(linkDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !filepath.IsAbs(canonical) {
		t.Errorf("expected absolute path, got %q", canonical)
	}

	if canonical != linkDir {
		t.Errorf("expected %q, got %q", linkDir, canonical)
	}
}

// TestCanonicalizeProjectID_AlreadyAbsolute は既に絶対パスの場合も正しく処理されることをテスト
func TestCanonicalizeProjectID_AlreadyAbsolute(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")

	if err := os.Mkdir(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}

	canonical, err := CanonicalizeProjectID(projectDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if canonical != projectDir {
		t.Errorf("expected %q, got %q", projectDir, canonical)
	}
}

// TestExpandTilde_Valid は有効なチルダパスが展開されることをテスト
func TestExpandTilde_Valid(t *testing.T) {
	path := "~/test/path"
	expanded, err := ExpandTilde(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expected := filepath.Join(home, "test", "path")
	if expanded != expected {
		t.Errorf("expected %q, got %q", expected, expanded)
	}
}

// TestExpandTilde_NoTilde はチルダがないパスがそのまま返されることをテスト
func TestExpandTilde_NoTilde(t *testing.T) {
	path := "/absolute/path"
	expanded, err := ExpandTilde(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if expanded != path {
		t.Errorf("expected %q, got %q", path, expanded)
	}
}

// TestExpandTilde_TildeOnly はチルダのみのパスが展開されることをテスト
func TestExpandTilde_TildeOnly(t *testing.T) {
	path := "~"
	expanded, err := ExpandTilde(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	if expanded != home {
		t.Errorf("expected %q, got %q", home, expanded)
	}
}

// TestExpandTilde_TildeUser は~user形式が展開されないことをテスト
func TestExpandTilde_TildeUser(t *testing.T) {
	path := "~otheruser/path"
	expanded, err := ExpandTilde(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ~user形式は未対応なのでそのまま返される
	if expanded != path {
		t.Errorf("expected %q (unchanged), got %q", path, expanded)
	}
}

// TestGetDefaultConfigPath はデフォルト設定ファイルパスが取得できることをテスト
func TestGetDefaultConfigPath(t *testing.T) {
	path, err := GetDefaultConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path == "" {
		t.Error("expected non-empty config path")
	}

	// ~/.local-mcp-memory/config.json の形式であることを確認
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %q", path)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expectedDir := filepath.Join(home, ".local-mcp-memory")
	expectedFile := filepath.Join(expectedDir, "config.json")

	if path != expectedFile {
		t.Errorf("expected %q, got %q", expectedFile, path)
	}
}

// TestGetDefaultDataDir はデフォルトデータディレクトリが取得できることをテスト
func TestGetDefaultDataDir(t *testing.T) {
	dir, err := GetDefaultDataDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dir == "" {
		t.Error("expected non-empty data dir")
	}

	// ~/.local-mcp-memory/data の形式であることを確認
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	expectedDir := filepath.Join(home, ".local-mcp-memory", "data")

	if dir != expectedDir {
		t.Errorf("expected %q, got %q", expectedDir, dir)
	}
}
