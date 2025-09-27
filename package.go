package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PackageHelper provides package management for ConfigHub resources
// Note: Package commands are experimental and require CONFIGHUB_EXPERIMENTAL=1
type PackageHelper struct {
	cub *ConfigHubClient
}

// PackageOptions contains options for package operations
type PackageOptions struct {
	SpaceID uuid.UUID         // Space to export from
	Where   string            // WHERE clause to filter resources
	Filter  string            // Filter name to apply
	Prefix  string            // Prefix for loaded resources
	Labels  map[string]string // Additional labels to apply
}

// PackageManifest represents the package manifest structure
type PackageManifest struct {
	Version     string       `json:"version,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	Description string       `json:"description,omitempty"`
	Spaces      []SpaceEntry `json:"spaces"`
	Units       []UnitEntry  `json:"units"`
	Links       []LinkEntry  `json:"links,omitempty"`
	Filters     []FilterEntry `json:"filters,omitempty"`
	Workers     []WorkerEntry `json:"workers,omitempty"`
	Targets     []TargetEntry `json:"targets,omitempty"`
}

// SpaceEntry represents a space in the manifest
type SpaceEntry struct {
	Slug       string `json:"slug"`
	DetailsLoc string `json:"details_loc"`
}

// UnitEntry represents a unit in the manifest
type UnitEntry struct {
	Slug        string `json:"slug"`
	SpaceSlug   string `json:"space_slug"`
	DetailsLoc  string `json:"details_loc"`
	UnitDataLoc string `json:"unit_data_loc"`
}

// LinkEntry represents a link in the manifest
type LinkEntry struct {
	Slug       string `json:"slug"`
	SpaceSlug  string `json:"space_slug"`
	FromUnit   string `json:"from_unit"`
	ToUnit     string `json:"to_unit"`
	DetailsLoc string `json:"details_loc"`
}

// FilterEntry represents a filter in the manifest
type FilterEntry struct {
	Slug       string `json:"slug"`
	SpaceSlug  string `json:"space_slug"`
	DetailsLoc string `json:"details_loc"`
}

// WorkerEntry represents a worker in the manifest
type WorkerEntry struct {
	Slug       string `json:"slug"`
	SpaceSlug  string `json:"space_slug"`
	DetailsLoc string `json:"details_loc"`
}

// TargetEntry represents a target in the manifest
type TargetEntry struct {
	Slug       string `json:"slug"`
	SpaceSlug  string `json:"space_slug"`
	DetailsLoc string `json:"details_loc"`
	Worker     string `json:"worker,omitempty"`
}

// NewPackageHelper creates a new package helper
func NewPackageHelper(cub *ConfigHubClient) *PackageHelper {
	return &PackageHelper{
		cub: cub,
	}
}

// CreatePackage exports ConfigHub resources to a package directory
// This wraps the `cub package create` command
func (p *PackageHelper) CreatePackage(dir string, opts PackageOptions) error {
	// Ensure experimental features are enabled
	env := append(os.Environ(), "CONFIGHUB_EXPERIMENTAL=1")

	args := []string{"package", "create", dir}

	// Add space if provided
	if opts.SpaceID != uuid.Nil {
		args = append(args, "--space", opts.SpaceID.String())
	}

	// Add where clause
	if opts.Where != "" {
		args = append(args, "--where", opts.Where)
	}

	// Add filter
	if opts.Filter != "" {
		args = append(args, "--filter", opts.Filter)
	}

	// Execute command
	cmd := exec.Command("cub", args...)
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("package create failed: %v\nStderr: %s", err, stderr.String())
	}

	// Add version info to manifest if not present
	manifestPath := filepath.Join(dir, "manifest.json")
	if err := p.enhanceManifest(manifestPath, opts); err != nil {
		// Non-critical error, just log it
		fmt.Printf("Warning: Could not enhance manifest: %v\n", err)
	}

	return nil
}

// LoadPackage imports a package from directory or URL
// This wraps the `cub package load` command
func (p *PackageHelper) LoadPackage(source string, prefix string) error {
	// Ensure experimental features are enabled
	env := append(os.Environ(), "CONFIGHUB_EXPERIMENTAL=1")

	args := []string{"package", "load", source}

	// Add prefix if provided
	if prefix != "" {
		args = append(args, "--prefix", prefix)
	}

	// Execute command
	cmd := exec.Command("cub", args...)
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("package load failed: %v\nStderr: %s", err, stderr.String())
	}

	return nil
}

// LoadPackageFromGitHub loads a package directly from a GitHub repository
func (p *PackageHelper) LoadPackageFromGitHub(owner, repo, path string, prefix string) error {
	// Construct GitHub raw URL
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s",
		owner, repo, strings.TrimPrefix(path, "/"))

	return p.LoadPackage(url, prefix)
}

// ValidatePackage checks if a package directory is valid
func (p *PackageHelper) ValidatePackage(dir string) error {
	// Check manifest exists
	manifestPath := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("manifest.json not found in package directory")
	}

	// Load and validate manifest
	manifest, err := p.LoadManifest(manifestPath)
	if err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// Check required directories exist
	for _, unit := range manifest.Units {
		unitDataPath := filepath.Join(dir, unit.UnitDataLoc)
		if _, err := os.Stat(unitDataPath); os.IsNotExist(err) {
			return fmt.Errorf("unit data file not found: %s", unit.UnitDataLoc)
		}
	}

	return nil
}

// LoadManifest loads a package manifest from file
func (p *PackageHelper) LoadManifest(path string) (*PackageManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var manifest PackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// enhanceManifest adds additional metadata to the manifest
func (p *PackageHelper) enhanceManifest(manifestPath string, opts PackageOptions) error {
	// Load existing manifest
	manifest, err := p.LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	// Add metadata if not present
	if manifest.CreatedAt.IsZero() {
		manifest.CreatedAt = time.Now()
	}

	if manifest.Description == "" && opts.SpaceID != uuid.Nil {
		manifest.Description = fmt.Sprintf("Package exported from space %s", opts.SpaceID)
	}

	// Write back
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(manifestPath, data, 0644)
}

// CreateVersionedPackage creates a package with version information
func (p *PackageHelper) CreateVersionedPackage(dir string, version string, opts PackageOptions) error {
	// Create the package
	if err := p.CreatePackage(dir, opts); err != nil {
		return err
	}

	// Update manifest with version
	manifestPath := filepath.Join(dir, "manifest.json")
	manifest, err := p.LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	manifest.Version = version
	manifest.Description = fmt.Sprintf("Version %s - %s", version, manifest.Description)

	// Write back
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(manifestPath, data, 0644)
}

// ListPackageContents lists the contents of a package
func (p *PackageHelper) ListPackageContents(dir string) (*PackageManifest, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	return p.LoadManifest(manifestPath)
}

// CloneEnvironment uses packages to clone an entire environment
func (p *PackageHelper) CloneEnvironment(sourceSpace uuid.UUID, targetPrefix string) error {
	// Create temporary directory for package
	tmpDir, err := os.MkdirTemp("", "confighub-clone-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Export source environment
	if err := p.CreatePackage(tmpDir, PackageOptions{
		SpaceID: sourceSpace,
	}); err != nil {
		return fmt.Errorf("failed to export source environment: %w", err)
	}

	// Load into target with prefix
	if err := p.LoadPackage(tmpDir, targetPrefix); err != nil {
		return fmt.Errorf("failed to load into target environment: %w", err)
	}

	return nil
}

// BackupSpace creates a timestamped backup package of a space
func (p *PackageHelper) BackupSpace(spaceID uuid.UUID, backupDir string) (string, error) {
	// Create timestamped directory
	timestamp := time.Now().Format("20060102-150405")
	packageDir := filepath.Join(backupDir, fmt.Sprintf("backup-%s", timestamp))

	// Create backup package
	if err := p.CreatePackage(packageDir, PackageOptions{
		SpaceID: spaceID,
	}); err != nil {
		return "", err
	}

	return packageDir, nil
}

// RestoreSpace restores a space from a backup package
func (p *PackageHelper) RestoreSpace(backupPath string, prefix string) error {
	// Validate package first
	if err := p.ValidatePackage(backupPath); err != nil {
		return fmt.Errorf("invalid backup package: %w", err)
	}

	// Restore with prefix to avoid conflicts
	if prefix == "" {
		prefix = fmt.Sprintf("restored-%d", time.Now().Unix())
	}

	return p.LoadPackage(backupPath, prefix)
}

// PublishPackage publishes a package to a git repository
func (p *PackageHelper) PublishPackage(packageDir string, repoURL string, message string) error {
	// Initialize git if needed
	if _, err := os.Stat(filepath.Join(packageDir, ".git")); os.IsNotExist(err) {
		cmd := exec.Command("git", "init")
		cmd.Dir = packageDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to initialize git: %w", err)
		}

		// Add remote
		cmd = exec.Command("git", "remote", "add", "origin", repoURL)
		cmd.Dir = packageDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	}

	// Add all files
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = packageDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add files: %w", err)
	}

	// Commit
	if message == "" {
		message = fmt.Sprintf("Package update - %s", time.Now().Format("2006-01-02 15:04:05"))
	}
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = packageDir
	if err := cmd.Run(); err != nil {
		// Check if there are no changes to commit
		statusCmd := exec.Command("git", "status", "--porcelain")
		statusCmd.Dir = packageDir
		output, _ := statusCmd.Output()
		if len(output) == 0 {
			return nil // No changes to commit
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push
	cmd = exec.Command("git", "push", "origin", "main")
	cmd.Dir = packageDir
	if err := cmd.Run(); err != nil {
		// Try force push for first push
		cmd = exec.Command("git", "push", "-u", "origin", "main", "--force")
		cmd.Dir = packageDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to push: %w", err)
		}
	}

	return nil
}

// FetchRemoteManifest fetches just the manifest from a remote package
func (p *PackageHelper) FetchRemoteManifest(url string) (*PackageManifest, error) {
	// Append manifest.json to URL if not already there
	if !strings.HasSuffix(url, "manifest.json") {
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		url += "manifest.json"
	}

	// Fetch manifest
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var manifest PackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}