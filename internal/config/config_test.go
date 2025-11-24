package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-edge-platform/os-image-composer/internal/utils/slice"
)

func TestMergeStringSlices(t *testing.T) {
	defaultSlice := []string{"a", "b", "c"}
	userSlice := []string{"c", "d", "e"}

	merged := mergeStringSlices(defaultSlice, userSlice)

	expectedLength := 5 // a, b, c, d, e (no duplicates)
	if len(merged) != expectedLength {
		t.Errorf("expected merged slice length %d, got %d", expectedLength, len(merged))
	}

	// Verify no duplicates
	itemMap := make(map[string]int)
	for _, item := range merged {
		itemMap[item]++
		if itemMap[item] > 1 {
			t.Errorf("found duplicate item '%s' in merged slice", item)
		}
	}

	// Verify all expected items are present
	expectedItems := []string{"a", "b", "c", "d", "e"}
	for _, expectedItem := range expectedItems {
		if itemMap[expectedItem] != 1 {
			t.Errorf("expected item '%s' to be present exactly once", expectedItem)
		}
	}
}

func TestMergeStringSlicesEmpty(t *testing.T) {
	// Both slices empty
	result := mergeStringSlices([]string{}, []string{})
	if len(result) != 0 {
		t.Errorf("expected empty result for two empty slices, got %d items", len(result))
	}

	// One slice empty
	slice1 := []string{"a", "b"}
	result = mergeStringSlices(slice1, []string{})
	if len(result) != 2 {
		t.Errorf("expected 2 items when second slice is empty, got %d", len(result))
	}

	result = mergeStringSlices([]string{}, slice1)
	if len(result) != 2 {
		t.Errorf("expected 2 items when first slice is empty, got %d", len(result))
	}
}

func TestMergeStringSlicesWithNils(t *testing.T) {
	slice1 := []string{"a", "b"}

	// This tests the actual behavior of mergeStringSlices with nil slices
	result := mergeStringSlices(nil, slice1)
	if len(result) != 2 {
		t.Errorf("expected 2 items when first slice is nil, got %d", len(result))
	}

	result = mergeStringSlices(slice1, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 items when second slice is nil, got %d", len(result))
	}
}

func TestEmptyUsersConfig(t *testing.T) {
	// Test template with no users
	template := &ImageTemplate{
		Image: ImageInfo{
			Name:    "test",
			Version: "1.0.0",
		},
		Target: TargetInfo{
			OS:        "azure-linux",
			Dist:      "azl3",
			Arch:      "x86_64",
			ImageType: "raw",
		},
		SystemConfig: SystemConfig{
			Name: "test-config",
			// No users configured
		},
	}

	// Test that empty users config works
	users := template.GetUsers()
	if len(users) != 0 {
		t.Errorf("expected 0 users for empty config, got %d", len(users))
	}

	if template.HasUsers() {
		t.Errorf("expected template to not have users")
	}

	nonExistentUser := template.GetUserByName("anyuser")
	if nonExistentUser != nil {
		t.Errorf("expected not to find any user in empty config")
	}
}

func TestMergeSystemConfigWithSecureBoot(t *testing.T) {
	defaultConfig := SystemConfig{
		Name: "default",
		Immutability: ImmutabilityConfig{
			Enabled:         true,
			SecureBootDBKey: "/default/keys/db.key",
			SecureBootDBCrt: "/default/certs/db.crt",
		},
		Packages: []string{"base-package"},
	}

	userConfig := SystemConfig{
		Name: "user",
		Immutability: ImmutabilityConfig{
			Enabled:         true,
			SecureBootDBKey: "/user/keys/custom.key",  // Override key
			SecureBootDBCer: "/user/certs/custom.cer", // Add new cer
			// Don't override crt - should keep default
		},
		Packages: []string{"user-package"},
	}

	merged := mergeSystemConfig(defaultConfig, userConfig)

	// Verify immutability merging
	if !merged.Immutability.Enabled {
		t.Errorf("expected merged immutability to be enabled")
	}

	if merged.Immutability.SecureBootDBKey != "/user/keys/custom.key" {
		t.Errorf("expected user secure boot key to override default")
	}

	if merged.Immutability.SecureBootDBCrt != "/default/certs/db.crt" {
		t.Errorf("expected default secure boot crt to be preserved")
	}

	if merged.Immutability.SecureBootDBCer != "/user/certs/custom.cer" {
		t.Errorf("expected user secure boot cer to be added")
	}
}

func TestLoadYAMLTemplateWithImmutability(t *testing.T) {
	// Create a temporary YAML file with immutability configuration under systemConfig
	yamlContent := `image:
  name: azl3-x86_64-edge
  version: "1.0.0"

target:
  os: azure-linux
  dist: azl3
  arch: x86_64
  imageType: raw

systemConfig:
  name: edge
  description: Default yml configuration for edge image
  immutability:
    enabled: true
  packages:
    - openssh-server
    - docker-ce
  kernel:
    version: "6.12"
    cmdline: "quiet splash"
`

	tmpFile, err := os.CreateTemp("", "test-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading
	template, err := LoadTemplate(tmpFile.Name(), true)
	if err != nil {
		t.Fatalf("failed to load YAML template: %v", err)
	}

	// Verify immutability configuration
	if !template.IsImmutabilityEnabled() {
		t.Errorf("expected immutability to be enabled, got %t", template.IsImmutabilityEnabled())
	}

	// Test direct access to systemConfig immutability
	if !template.SystemConfig.IsImmutabilityEnabled() {
		t.Errorf("expected systemConfig immutability to be enabled, got %t", template.SystemConfig.IsImmutabilityEnabled())
	}
}

func TestMergeSystemConfigWithImmutability(t *testing.T) {
	defaultConfig := SystemConfig{
		Name:         "default",
		Immutability: ImmutabilityConfig{Enabled: true},
		Packages:     []string{"base-package"},
	}

	userConfig := SystemConfig{
		Name:         "user",
		Immutability: ImmutabilityConfig{Enabled: false},
		Packages:     []string{"user-package"},
	}

	merged := mergeSystemConfig(defaultConfig, userConfig)

	if merged.Immutability.Enabled != false {
		t.Errorf("expected merged immutability to be false (user override), got %t", merged.Immutability.Enabled)
	}

	if merged.Name != "user" {
		t.Errorf("expected merged name to be 'user', got %s", merged.Name)
	}
}

func TestTemplateHelperMethodsWithImmutability(t *testing.T) {
	template := &ImageTemplate{
		Image: ImageInfo{
			Name:    "test-image",
			Version: "1.0.0",
		},
		Target: TargetInfo{
			OS:        "azure-linux",
			Dist:      "azl3",
			Arch:      "x86_64",
			ImageType: "raw",
		},
		SystemConfig: SystemConfig{
			Name:         "test-config",
			Description:  "Test configuration",
			Immutability: ImmutabilityConfig{Enabled: true},
			Packages:     []string{"package1", "package2"},
			Kernel: KernelConfig{
				Version: "6.12",
				Cmdline: "quiet",
			},
		},
	}

	// Test immutability access methods
	if !template.IsImmutabilityEnabled() {
		t.Errorf("expected immutability to be enabled, got %t", template.IsImmutabilityEnabled())
	}

	immutabilityConfig := template.GetImmutability()
	if !immutabilityConfig.Enabled {
		t.Errorf("expected immutability config to be enabled, got %t", immutabilityConfig.Enabled)
	}

	// Test systemConfig direct access
	if !template.SystemConfig.IsImmutabilityEnabled() {
		t.Errorf("expected systemConfig immutability to be enabled, got %t", template.SystemConfig.IsImmutabilityEnabled())
	}
}

func TestTemplateHelperMethodsWithUsers(t *testing.T) {
	template := &ImageTemplate{
		Image: ImageInfo{
			Name:    "test-image",
			Version: "1.0.0",
		},
		Target: TargetInfo{
			OS:        "azure-linux",
			Dist:      "azl3",
			Arch:      "x86_64",
			ImageType: "raw",
		},
		SystemConfig: SystemConfig{
			Name:        "test-config",
			Description: "Test configuration",
			Users: []UserConfig{
				{Name: "testuser", Password: "testpass", HashAlgo: "sha512", Sudo: true},
				{Name: "admin", Password: "$6$test$hash", Groups: []string{"wheel"}, PasswordMaxAge: 365},
			},
			Packages: []string{"package1", "package2"},
			Kernel: KernelConfig{
				Version: "6.12",
				Cmdline: "quiet",
			},
		},
	}

	// Test users access methods
	users := template.GetUsers()
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	testUser := template.GetUserByName("testuser")
	if testUser == nil {
		t.Errorf("expected to find testuser")
	} else {
		if !testUser.Sudo {
			t.Errorf("expected testuser to have sudo privileges")
		}
		if testUser.HashAlgo != "sha512" {
			t.Errorf("expected testuser hash_algo 'sha512', got %s", testUser.HashAlgo)
		}
	}

	// Test non-existent user
	nonExistentUser := template.GetUserByName("nonexistent")
	if nonExistentUser != nil {
		t.Errorf("expected not to find nonexistent user")
	}

	if !template.HasUsers() {
		t.Errorf("expected template to have users")
	}

	// Test systemConfig direct access
	if !template.SystemConfig.HasUsers() {
		t.Errorf("expected systemConfig to have users")
	}

	adminUser := template.SystemConfig.GetUserByName("admin")
	if adminUser == nil {
		t.Errorf("expected to find admin user via systemConfig")
	} else {
		if adminUser.PasswordMaxAge != 365 {
			t.Errorf("expected admin passwordMaxAge to be 365, got %d", adminUser.PasswordMaxAge)
		}
	}
}

func TestMergeSystemConfigWithUsers(t *testing.T) {
	defaultConfig := SystemConfig{
		Name: "default",
		Users: []UserConfig{
			{Name: "defaultuser", Password: "defaultpass", HashAlgo: "sha512"},
			{Name: "shared", Password: "defaultshared", HashAlgo: "sha256", Groups: []string{"default"}},
		},
		Packages: []string{"base-package"},
	}

	userConfig := SystemConfig{
		Name: "user",
		Users: []UserConfig{
			{Name: "newuser", Password: "newpass", HashAlgo: "bcrypt"},
			{Name: "shared", Password: "usershared", HashAlgo: "sha512", Groups: []string{"user", "admin"}, PasswordMaxAge: 180},
		},
		Packages: []string{"user-package"},
	}

	merged := mergeSystemConfig(defaultConfig, userConfig)

	// Test user merge
	if len(merged.Users) != 3 {
		t.Errorf("expected 3 merged users, got %d", len(merged.Users))
	}

	// Find shared user to test merging
	var sharedUser *UserConfig
	for i := range merged.Users {
		if merged.Users[i].Name == "shared" {
			sharedUser = &merged.Users[i]
			break
		}
	}

	if sharedUser == nil {
		t.Errorf("expected to find shared user in merged config")
	} else {
		if sharedUser.Password != "usershared" {
			t.Errorf("expected shared user password 'usershared', got '%s'", sharedUser.Password)
		}
		if sharedUser.HashAlgo != "sha512" {
			t.Errorf("expected shared user hash algo 'sha512', got '%s'", sharedUser.HashAlgo)
		}
		if sharedUser.PasswordMaxAge != 180 {
			t.Errorf("expected shared user password max age 180, got %d", sharedUser.PasswordMaxAge)
		}
		if len(sharedUser.Groups) != 3 { // default, user, admin merged
			t.Errorf("expected 3 merged groups for shared user, got %d", len(sharedUser.Groups))
		}
	}

	// Verify expected groups are present
	expectedGroups := []string{"default", "user", "admin"}
	groupMap := make(map[string]bool)
	for _, group := range sharedUser.Groups {
		groupMap[group] = true
	}
	for _, expectedGroup := range expectedGroups {
		if !groupMap[expectedGroup] {
			t.Errorf("expected group '%s' to be in merged groups", expectedGroup)
		}
	}
}

func TestUnsupportedFileFormat(t *testing.T) {
	// Create a temporary file with unsupported extension
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("some content"); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading should fail
	_, err = LoadTemplate(tmpFile.Name(), false)
	if err == nil {
		t.Errorf("expected error for unsupported file format")
	}
	if !strings.Contains(err.Error(), "unsupported file format") {
		t.Errorf("expected unsupported file format error, got: %v", err)
	}
}

func TestEmptySystemConfig(t *testing.T) {
	// Test template with empty system config
	template := &ImageTemplate{
		Image: ImageInfo{
			Name:    "test",
			Version: "1.0.0",
		},
		Target: TargetInfo{
			OS:        "azure-linux",
			Dist:      "azl3",
			Arch:      "x86_64",
			ImageType: "raw",
		},
		// Empty SystemConfig
		SystemConfig: SystemConfig{},
	}

	// Test that empty config still works
	packages := template.GetPackages()
	if len(packages) != 0 {
		t.Errorf("expected 0 packages for empty config, got %d", len(packages))
	}

	configName := template.GetSystemConfigName()
	if configName != "" {
		t.Errorf("expected empty config name, got %s", configName)
	}
}

func TestAllSupportedProviders(t *testing.T) {
	testCases := []struct {
		os       string
		dist     string
		expected string
		version  string
	}{
		{"azure-linux", "azl3", "AzureLinux3", "3"},
		{"emt", "emt3", "EMT3.0", "3.0"},
		{"elxr", "elxr12", "eLxr12", "12"},
	}

	for _, tc := range testCases {
		template := &ImageTemplate{
			Target: TargetInfo{
				OS:        tc.os,
				Dist:      tc.dist,
				Arch:      "x86_64",
				ImageType: "iso",
			},
			SystemConfig: SystemConfig{
				Name:     "test",
				Packages: []string{"test-package"},
				Kernel:   KernelConfig{Version: "6.12"},
			},
		}

		// Test provider name
		providerName := template.GetProviderName()
		if providerName != tc.expected {
			t.Errorf("for %s/%s expected provider '%s', got '%s'", tc.os, tc.dist, tc.expected, providerName)
		}

		// Test version
		version := template.GetDistroVersion()
		if version != tc.version {
			t.Errorf("for %s/%s expected version '%s', got '%s'", tc.os, tc.dist, tc.version, version)
		}
	}
}

func TestDiskAndSystemConfigGetters(t *testing.T) {
	template := &ImageTemplate{
		Image: ImageInfo{
			Name:    "test-image",
			Version: "1.0.0",
		},
		Target: TargetInfo{
			OS:        "azure-linux",
			Dist:      "azl3",
			Arch:      "x86_64",
			ImageType: "raw",
		},
		Disk: DiskConfig{
			Name: "test-disk",
			Size: "4GiB",
			Partitions: []PartitionInfo{
				{
					ID:         "root",
					FsType:     "ext4",
					Start:      "1MiB",
					End:        "0",
					MountPoint: "/",
				},
			},
		},
		SystemConfig: SystemConfig{
			Name: "test-config",
			Bootloader: Bootloader{
				BootType: "efi",
				Provider: "grub2",
			},
			Packages: []string{"package1", "package2"},
			Kernel: KernelConfig{
				Version: "6.12",
				Cmdline: "quiet splash",
			},
		},
	}

	// Test disk config getter
	diskConfig := template.GetDiskConfig()
	if diskConfig.Name != "test-disk" {
		t.Errorf("expected disk name 'test-disk', got %s", diskConfig.Name)
	}
	if diskConfig.Size != "4GiB" {
		t.Errorf("expected disk size '4GiB', got %s", diskConfig.Size)
	}
	if len(diskConfig.Partitions) != 1 {
		t.Errorf("expected 1 partition, got %d", len(diskConfig.Partitions))
	}

	// Test system config getter
	systemConfig := template.GetSystemConfig()
	if systemConfig.Name != "test-config" {
		t.Errorf("expected system config name 'test-config', got %s", systemConfig.Name)
	}

	// Test bootloader config getter
	bootloaderConfig := template.GetBootloaderConfig()
	if bootloaderConfig.BootType != "efi" {
		t.Errorf("expected bootloader type 'efi', got %s", bootloaderConfig.BootType)
	}
	if bootloaderConfig.Provider != "grub2" {
		t.Errorf("expected bootloader provider 'grub2', got %s", bootloaderConfig.Provider)
	}

	// Test individual field access
	packages := template.GetPackages()
	if len(packages) != 2 {
		t.Errorf("expected 2 packages, got %d", len(packages))
	}

	// Test kernel extraction
	kernel := template.GetKernel()
	if kernel.Version != "6.12" {
		t.Errorf("expected kernel version '6.12', got %s", kernel.Version)
	}

	// Test system config name extraction
	configName := template.GetSystemConfigName()
	if configName != "test-config" {
		t.Errorf("expected config name 'test-config', got %s", configName)
	}
}

func TestSecureBootHelperMethods(t *testing.T) {
	template := &ImageTemplate{
		Image: ImageInfo{
			Name:    "test-image",
			Version: "1.0.0",
		},
		Target: TargetInfo{
			OS:        "azure-linux",
			Dist:      "azl3",
			Arch:      "x86_64",
			ImageType: "raw",
		},
		SystemConfig: SystemConfig{
			Name:        "test-config",
			Description: "Test configuration with secure boot",
			Immutability: ImmutabilityConfig{
				Enabled:         true,
				SecureBootDBKey: "/test/keys/db.key",
				SecureBootDBCrt: "/test/certs/db.crt",
				SecureBootDBCer: "/test/certs/db.cer",
			},
		},
	}

	// Test ImmutabilityConfig helper methods
	immutabilityConfig := template.GetImmutability()
	if !immutabilityConfig.HasSecureBootDBConfig() {
		t.Errorf("expected immutability config to have secure boot DB config")
	}

	if !immutabilityConfig.HasSecureBootDBKey() {
		t.Errorf("expected immutability config to have secure boot DB key")
	}

	if !immutabilityConfig.HasSecureBootDBCrt() {
		t.Errorf("expected immutability config to have secure boot DB crt")
	}

	if !immutabilityConfig.HasSecureBootDBCer() {
		t.Errorf("expected immutability config to have secure boot DB cer")
	}

	// Test path retrieval methods
	if keyPath := immutabilityConfig.GetSecureBootDBKeyPath(); keyPath != "/test/keys/db.key" {
		t.Errorf("expected key path '/test/keys/db.key', got '%s'", keyPath)
	}

	if crtPath := immutabilityConfig.GetSecureBootDBCrtPath(); crtPath != "/test/certs/db.crt" {
		t.Errorf("expected crt path '/test/certs/db.crt', got '%s'", crtPath)
	}

	if cerPath := immutabilityConfig.GetSecureBootDBCerPath(); cerPath != "/test/certs/db.cer" {
		t.Errorf("expected cer path '/test/certs/db.cer', got '%s'", cerPath)
	}

	// Test SystemConfig access methods
	systemConfig := template.SystemConfig
	if !systemConfig.HasSecureBootDBConfig() {
		t.Errorf("expected systemConfig to have secure boot DB config")
	}

	if keyPath := systemConfig.GetSecureBootDBKeyPath(); keyPath != "/test/keys/db.key" {
		t.Errorf("expected systemConfig key path '/test/keys/db.key', got '%s'", keyPath)
	}

	if crtPath := systemConfig.GetSecureBootDBCrtPath(); crtPath != "/test/certs/db.crt" {
		t.Errorf("expected systemConfig crt path '/test/certs/db.crt', got '%s'", crtPath)
	}

	if cerPath := systemConfig.GetSecureBootDBCerPath(); cerPath != "/test/certs/db.cer" {
		t.Errorf("expected systemConfig cer path '/test/certs/db.cer', got '%s'", cerPath)
	}

	// Test ImageTemplate secure boot helpers
	expectedKeyPath := "/test/keys/db.key"
	if keyPath := template.GetSecureBootDBKeyPath(); keyPath != expectedKeyPath {
		t.Errorf("expected secure boot key path '%s', got '%s'", expectedKeyPath, keyPath)
	}

	expectedCrtPath := "/test/certs/db.crt"
	if crtPath := template.GetSecureBootDBCrtPath(); crtPath != expectedCrtPath {
		t.Errorf("expected secure boot crt path '%s', got '%s'", expectedCrtPath, crtPath)
	}

	expectedCerPath := "/test/certs/db.cer"
	if cerPath := template.GetSecureBootDBCerPath(); cerPath != expectedCerPath {
		t.Errorf("expected secure boot cer path '%s', got '%s'", expectedCerPath, cerPath)
	}

	if !template.HasSecureBootDBConfig() {
		t.Errorf("expected template to have secure boot DB config")
	}
}

func TestSecureBootWithoutConfig(t *testing.T) {
	template := &ImageTemplate{
		Image: ImageInfo{
			Name:    "test-image",
			Version: "1.0.0",
		},
		Target: TargetInfo{
			OS:        "azure-linux",
			Dist:      "azl3",
			Arch:      "x86_64",
			ImageType: "raw",
		},
		SystemConfig: SystemConfig{
			Name:        "test-config",
			Description: "Test configuration without secure boot",
			Immutability: ImmutabilityConfig{
				Enabled: true,
				// No secure boot fields set
			},
		},
	}

	// Test that methods work correctly when no secure boot config is provided
	if template.HasSecureBootDBConfig() {
		t.Errorf("expected template to not have secure boot DB config")
	}

	immutabilityConfig := template.GetImmutability()
	if immutabilityConfig.HasSecureBootDBConfig() {
		t.Errorf("expected immutability config to not have secure boot DB config")
	}

	if immutabilityConfig.HasSecureBootDBKey() {
		t.Errorf("expected immutability config to not have secure boot DB key")
	}

	if immutabilityConfig.HasSecureBootDBCrt() {
		t.Errorf("expected immutability config to not have secure boot DB crt")
	}

	if immutabilityConfig.HasSecureBootDBCer() {
		t.Errorf("expected immutability config to not have secure boot DB cer")
	}

	// Test that path methods return empty strings
	if keyPath := template.GetSecureBootDBKeyPath(); keyPath != "" {
		t.Errorf("expected empty key path, got '%s'", keyPath)
	}

	if crtPath := template.GetSecureBootDBCrtPath(); crtPath != "" {
		t.Errorf("expected empty crt path, got '%s'", crtPath)
	}

	if cerPath := template.GetSecureBootDBCerPath(); cerPath != "" {
		t.Errorf("expected empty cer path, got '%s'", cerPath)
	}
}

func TestPartialSecureBootConfig(t *testing.T) {
	template := &ImageTemplate{
		SystemConfig: SystemConfig{
			Immutability: ImmutabilityConfig{
				Enabled:         true,
				SecureBootDBKey: "/test/keys/db.key",
				// Only key is set, no certificates
			},
		},
	}

	immutabilityConfig := template.GetImmutability()

	// Should have config because key is set
	if !immutabilityConfig.HasSecureBootDBConfig() {
		t.Errorf("expected immutability config to have secure boot DB config")
	}

	// Should have key
	if !immutabilityConfig.HasSecureBootDBKey() {
		t.Errorf("expected immutability config to have secure boot DB key")
	}

	// Should not have certificates
	if immutabilityConfig.HasSecureBootDBCrt() {
		t.Errorf("expected immutability config to not have secure boot DB crt")
	}

	if immutabilityConfig.HasSecureBootDBCer() {
		t.Errorf("expected immutability config to not have secure boot DB cer")
	}
}

func TestDiskConfigValidation(t *testing.T) {
	testCases := []struct {
		name     string
		disk     DiskConfig
		expected bool // whether it should be considered empty
	}{
		{
			name:     "empty disk config",
			disk:     DiskConfig{},
			expected: true,
		},
		{
			name: "disk with only name",
			disk: DiskConfig{
				Name: "test-disk",
			},
			expected: false,
		},
		{
			name: "disk with full configuration",
			disk: DiskConfig{
				Name:               "main-disk",
				Size:               "20GiB",
				PartitionTableType: "gpt",
				Partitions: []PartitionInfo{
					{
						ID:         "boot",
						Name:       "EFI Boot",
						Type:       "esp",
						FsType:     "fat32",
						Start:      "1MiB",
						End:        "513MiB",
						MountPoint: "/boot/efi",
						Flags:      []string{"boot"},
					},
					{
						ID:         "root",
						Name:       "Root",
						Type:       "linux-root-amd64",
						FsType:     "ext4",
						Start:      "513MiB",
						End:        "0",
						MountPoint: "/",
					},
				},
			},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isEmpty := isEmptyDiskConfig(tc.disk)
			if isEmpty != tc.expected {
				t.Errorf("expected isEmptyDiskConfig to be %t, got %t", tc.expected, isEmpty)
			}
		})
	}
}

func TestPartitionInfoFields(t *testing.T) {
	template := &ImageTemplate{
		Disk: DiskConfig{
			Name:               "test-disk",
			Size:               "10GiB",
			PartitionTableType: "gpt",
			Partitions: []PartitionInfo{
				{
					ID:           "efi",
					Name:         "EFI System",
					Type:         "esp",
					TypeGUID:     "C12A7328-F81F-11D2-BA4B-00A0C93EC93B",
					FsType:       "fat32",
					Start:        "1MiB",
					End:          "513MiB",
					MountPoint:   "/boot/efi",
					MountOptions: "defaults",
					Flags:        []string{"boot", "esp"},
				},
				{
					ID:       "swap",
					Name:     "Swap",
					Type:     "swap",
					TypeGUID: "0657FD6D-A4AB-43C4-84E5-0933C84B4F4F",
					FsType:   "swap",
					Start:    "513MiB",
					End:      "2GiB",
				},
				{
					ID:         "root",
					Name:       "Root",
					Type:       "linux-root-amd64",
					TypeGUID:   "4F68BCE3-E8CD-4DB1-96E7-FBCAF984B709",
					FsType:     "ext4",
					Start:      "2GiB",
					End:        "0",
					MountPoint: "/",
				},
			},
		},
	}

	diskConfig := template.GetDiskConfig()

	// Verify partition count
	if len(diskConfig.Partitions) != 3 {
		t.Errorf("expected 3 partitions, got %d", len(diskConfig.Partitions))
	}

	// Verify EFI partition
	efiPartition := diskConfig.Partitions[0]
	if efiPartition.ID != "efi" {
		t.Errorf("expected EFI partition ID 'efi', got '%s'", efiPartition.ID)
	}
	if len(efiPartition.Flags) != 2 {
		t.Errorf("expected 2 flags for EFI partition, got %d", len(efiPartition.Flags))
	}
	if efiPartition.TypeGUID != "C12A7328-F81F-11D2-BA4B-00A0C93EC93B" {
		t.Errorf("expected EFI TypeGUID, got '%s'", efiPartition.TypeGUID)
	}
	if efiPartition.Start != "1MiB" {
		t.Errorf("expected EFI start '1MiB', got '%s'", efiPartition.Start)
	}
	if efiPartition.End != "513MiB" {
		t.Errorf("expected EFI end '513MiB', got '%s'", efiPartition.End)
	}
	if efiPartition.MountOptions != "defaults" {
		t.Errorf("expected EFI mount options 'defaults', got '%s'", efiPartition.MountOptions)
	}

	// Verify swap partition
	swapPartition := diskConfig.Partitions[1]
	if swapPartition.FsType != "swap" {
		t.Errorf("expected swap filesystem type, got '%s'", swapPartition.FsType)
	}
	if swapPartition.MountPoint != "" {
		t.Errorf("expected empty mount point for swap, got '%s'", swapPartition.MountPoint)
	}
	if swapPartition.Start != "513MiB" {
		t.Errorf("expected swap start '513MiB', got '%s'", swapPartition.Start)
	}
	if swapPartition.End != "2GiB" {
		t.Errorf("expected swap end '2GiB', got '%s'", swapPartition.End)
	}

	// Verify root partition
	rootPartition := diskConfig.Partitions[2]
	if rootPartition.MountPoint != "/" {
		t.Errorf("expected root mount point '/', got '%s'", rootPartition.MountPoint)
	}
	if rootPartition.Start != "2GiB" {
		t.Errorf("expected root start '2GiB', got '%s'", rootPartition.Start)
	}
	if rootPartition.End != "0" {
		t.Errorf("expected root end '0' (end of disk), got '%s'", rootPartition.End)
	}
}

func TestArtifactInfo(t *testing.T) {
	template := &ImageTemplate{
		Disk: DiskConfig{
			Name: "test-disk",
			Artifacts: []ArtifactInfo{
				{Type: "raw", Compression: "none"},
				{Type: "qcow2", Compression: "gzip"},
				{Type: "vmdk", Compression: "lz4"},
			},
		},
	}

	artifacts := template.GetDiskConfig().Artifacts
	if len(artifacts) != 3 {
		t.Errorf("expected 3 artifacts, got %d", len(artifacts))
	}

	// Test artifact types and compression
	expectedArtifacts := []struct {
		Type        string
		Compression string
	}{
		{"raw", "none"},
		{"qcow2", "gzip"},
		{"vmdk", "lz4"},
	}

	for i, expected := range expectedArtifacts {
		if artifacts[i].Type != expected.Type {
			t.Errorf("artifact %d: expected type '%s', got '%s'", i, expected.Type, artifacts[i].Type)
		}
		if artifacts[i].Compression != expected.Compression {
			t.Errorf("artifact %d: expected compression '%s', got '%s'", i, expected.Compression, artifacts[i].Compression)
		}
	}
}

func TestAdditionalFileInfo(t *testing.T) {
	template := &ImageTemplate{
		SystemConfig: SystemConfig{
			Name: "test-config",
			AdditionalFiles: []AdditionalFileInfo{
				{Local: "/host/config.conf", Final: "/etc/app/config.conf"},
				{Local: "/host/script.sh", Final: "/usr/local/bin/script.sh"},
				{Local: "/host/certs/ca.crt", Final: "/etc/ssl/certs/ca.crt"},
			},
		},
	}

	additionalFiles := template.GetSystemConfig().AdditionalFiles
	if len(additionalFiles) != 3 {
		t.Errorf("expected 3 additional files, got %d", len(additionalFiles))
	}

	// Test file mappings
	expectedFiles := []struct {
		Local string
		Final string
	}{
		{"/host/config.conf", "/etc/app/config.conf"},
		{"/host/script.sh", "/usr/local/bin/script.sh"},
		{"/host/certs/ca.crt", "/etc/ssl/certs/ca.crt"},
	}

	for i, expected := range expectedFiles {
		if additionalFiles[i].Local != expected.Local {
			t.Errorf("file %d: expected local path '%s', got '%s'", i, expected.Local, additionalFiles[i].Local)
		}
		if additionalFiles[i].Final != expected.Final {
			t.Errorf("file %d: expected final path '%s', got '%s'", i, expected.Final, additionalFiles[i].Final)
		}
	}
}

func TestMergeUserConfig(t *testing.T) {
	defaultUser := UserConfig{
		Name:           "user",
		Password:       "default",
		HashAlgo:       "sha512",
		PasswordMaxAge: 90,
		Groups:         []string{"group1"},
	}

	// Test override
	userUser := UserConfig{
		Name:     "user",
		Password: "newpassword",
		Groups:   []string{"group2"},
		Sudo:     true,
	}

	merged := mergeUserConfig(defaultUser, userUser)

	if merged.Password != "newpassword" {
		t.Errorf("Expected password newpassword, got %s", merged.Password)
	}
	if merged.HashAlgo != "sha512" {
		t.Errorf("Expected hash algo sha512, got %s", merged.HashAlgo)
	}
	if !merged.Sudo {
		t.Errorf("Expected sudo true")
	}
	if !slice.Contains(merged.Groups, "group1") || !slice.Contains(merged.Groups, "group2") {
		t.Errorf("Expected groups to contain group1 and group2, got %v", merged.Groups)
	}

	// Test pre-hashed password
	userUserHashed := UserConfig{
		Name:     "user",
		Password: "$6$hash",
	}

	mergedHashed := mergeUserConfig(defaultUser, userUserHashed)
	if mergedHashed.HashAlgo != "" {
		t.Errorf("Expected empty hash algo for pre-hashed password, got %s", mergedHashed.HashAlgo)
	}
}

func TestMergeAdditionalFiles(t *testing.T) {
	defaultFiles := []AdditionalFileInfo{
		{Local: "default1", Final: "/etc/file1"},
		{Local: "default2", Final: "/etc/file2"},
	}
	userFiles := []AdditionalFileInfo{
		{Local: "user1", Final: "/etc/file1"}, // Override
		{Local: "user3", Final: "/etc/file3"}, // New
	}

	merged := mergeAdditionalFiles(defaultFiles, userFiles)

	if len(merged) != 3 {
		t.Errorf("Expected 3 files, got %d", len(merged))
	}

	fileMap := make(map[string]AdditionalFileInfo)
	for _, f := range merged {
		fileMap[f.Final] = f
	}

	if fileMap["/etc/file1"].Local != "user1" {
		t.Errorf("Expected /etc/file1 to be user1, got %s", fileMap["/etc/file1"].Local)
	}
	if fileMap["/etc/file2"].Local != "default2" {
		t.Errorf("Expected /etc/file2 to be default2, got %s", fileMap["/etc/file2"].Local)
	}
	if fileMap["/etc/file3"].Local != "user3" {
		t.Errorf("Expected /etc/file3 to be user3, got %s", fileMap["/etc/file3"].Local)
	}
}

func TestMergePackages(t *testing.T) {
	p1 := []string{"pkg1", "pkg2"}
	p2 := []string{"pkg2", "pkg3"}
	merged := mergePackages(p1, p2)

	if len(merged) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(merged))
	}
}

func TestMergeKernelConfig(t *testing.T) {
	defaultKernel := KernelConfig{
		Version:  "1.0",
		Cmdline:  "default",
		Packages: []string{"kernel-default"},
	}
	userKernel := KernelConfig{
		Version:            "2.0",
		EnableExtraModules: "true",
	}

	merged := mergeKernelConfig(defaultKernel, userKernel)

	if merged.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", merged.Version)
	}
	if merged.Cmdline != "default" {
		t.Errorf("Expected cmdline default, got %s", merged.Cmdline)
	}
	if merged.EnableExtraModules != "true" {
		t.Errorf("Expected enableExtraModules true, got %s", merged.EnableExtraModules)
	}
	if len(merged.Packages) != 1 || merged.Packages[0] != "kernel-default" {
		t.Errorf("Expected packages [kernel-default], got %v", merged.Packages)
	}
}

func TestLoadProviderRepoConfig(t *testing.T) {
	// Setup temporary config directory
	tempDir := t.TempDir()

	// Save original global config
	originalGlobal := Global()
	// Restore original global config after test
	defer SetGlobal(originalGlobal)

	// Set new global config with temp dir
	newGlobal := DefaultGlobalConfig()
	newGlobal.ConfigDir = tempDir
	SetGlobal(newGlobal)

	// Create directory structure
	// config/osv/testos/testdist/providerconfigs/repo.yml
	osDistDir := filepath.Join(tempDir, "osv", "testos", "testdist")
	providerConfigDir := filepath.Join(osDistDir, "providerconfigs")
	if err := os.MkdirAll(providerConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create directory structure: %v", err)
	}

	// Create repo.yml
	repoConfigContent := `
repositories:
  - name: test-repo
    type: rpm
    baseURL: http://example.com/repo
    gpgKey: http://example.com/key
    enabled: true
`
	repoConfigFile := filepath.Join(providerConfigDir, "repo.yml")
	if err := os.WriteFile(repoConfigFile, []byte(repoConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write repo config file: %v", err)
	}

	// Test LoadProviderRepoConfig
	repos, err := LoadProviderRepoConfig("testos", "testdist")
	if err != nil {
		t.Fatalf("LoadProviderRepoConfig failed: %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("Expected 1 repository, got %d", len(repos))
	}
	if repos[0].Name != "test-repo" {
		t.Errorf("Expected repo name 'test-repo', got '%s'", repos[0].Name)
	}
}

func TestToRepoConfigData(t *testing.T) {
	// Test RPM repo
	rpmRepo := ProviderRepoConfig{
		Name:    "rpm-repo",
		Type:    "rpm",
		BaseURL: "http://example.com/rpm/{arch}",
		GPGKey:  "key.gpg",
		Enabled: true,
	}

	repoType, name, url, gpgKey, _, _, _, _, _, _, _, _, enabled := rpmRepo.ToRepoConfigData("x86_64")

	if repoType != "rpm" {
		t.Errorf("Expected type rpm, got %s", repoType)
	}
	if name != "rpm-repo" {
		t.Errorf("Expected name rpm-repo, got %s", name)
	}
	if url != "http://example.com/rpm/x86_64" {
		t.Errorf("Expected url http://example.com/rpm/x86_64, got %s", url)
	}
	// Relative GPG key should be combined with URL
	expectedGpgKey := "http://example.com/rpm/x86_64/key.gpg"
	if gpgKey != expectedGpgKey {
		t.Errorf("Expected gpgKey %s, got %s", expectedGpgKey, gpgKey)
	}
	if !enabled {
		t.Errorf("Expected enabled true")
	}

	// Test DEB repo
	debRepo := ProviderRepoConfig{
		Name:      "deb-repo",
		Type:      "deb",
		BaseURL:   "http://example.com/deb",
		PbGPGKey:  "http://example.com/key.gpg",
		PkgPrefix: "prefix",
		Enabled:   true,
	}

	repoType, name, url, gpgKey, _, _, pkgPrefix, _, _, _, _, _, _ := debRepo.ToRepoConfigData("amd64")

	if repoType != "deb" {
		t.Errorf("Expected type deb, got %s", repoType)
	}
	if url != "http://example.com/deb/binary-amd64/Packages.gz" {
		t.Errorf("Expected url http://example.com/deb/binary-amd64/Packages.gz, got %s", url)
	}
	if gpgKey != "http://example.com/key.gpg" {
		t.Errorf("Expected gpgKey http://example.com/key.gpg, got %s", gpgKey)
	}
	if pkgPrefix != "prefix" {
		t.Errorf("Expected pkgPrefix prefix, got %s", pkgPrefix)
	}
}

func TestGetInitramfsTemplate(t *testing.T) {
	tempDir := t.TempDir()
	templateFile := filepath.Join(tempDir, "template.yml")
	initrdFile := filepath.Join(tempDir, "initrd.template")

	if err := os.WriteFile(initrdFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create initrd file: %v", err)
	}

	// Test absolute path
	tmpl := &ImageTemplate{
		SystemConfig: SystemConfig{
			Initramfs: Initramfs{
				Template: initrdFile,
			},
		},
	}

	path, err := tmpl.GetInitramfsTemplate()
	if err != nil {
		t.Fatalf("GetInitramfsTemplate failed with absolute path: %v", err)
	}
	if path != initrdFile {
		t.Errorf("Expected path %s, got %s", initrdFile, path)
	}

	// Test relative path
	tmplRelative := &ImageTemplate{
		SystemConfig: SystemConfig{
			Initramfs: Initramfs{
				Template: "initrd.template",
			},
		},
		PathList: []string{templateFile},
	}

	path, err = tmplRelative.GetInitramfsTemplate()
	if err != nil {
		t.Fatalf("GetInitramfsTemplate failed with relative path: %v", err)
	}
	if path != initrdFile {
		t.Errorf("Expected path %s, got %s", initrdFile, path)
	}
}

func TestGetAdditionalFileInfo(t *testing.T) {
	tempDir := t.TempDir()
	templateFile := filepath.Join(tempDir, "template.yml")
	localFile := filepath.Join(tempDir, "local.txt")

	if err := os.WriteFile(localFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create local file: %v", err)
	}

	// Test absolute path
	tmpl := &ImageTemplate{
		SystemConfig: SystemConfig{
			AdditionalFiles: []AdditionalFileInfo{
				{Local: localFile, Final: "/etc/final.txt"},
			},
		},
	}

	files := tmpl.GetAdditionalFileInfo()
	if len(files) != 1 {
		t.Errorf("Expected 1 additional file, got %d", len(files))
	}
	if files[0].Local != localFile {
		t.Errorf("Expected local path %s, got %s", localFile, files[0].Local)
	}

	// Test relative path
	tmplRelative := &ImageTemplate{
		SystemConfig: SystemConfig{
			AdditionalFiles: []AdditionalFileInfo{
				{Local: "local.txt", Final: "/etc/final.txt"},
			},
		},
		PathList: []string{templateFile},
	}

	files = tmplRelative.GetAdditionalFileInfo()
	if len(files) != 1 {
		t.Errorf("Expected 1 additional file, got %d", len(files))
	}
	if files[0].Local != localFile {
		t.Errorf("Expected local path %s, got %s", localFile, files[0].Local)
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	// Setup temporary config directory
	tempDir := t.TempDir()

	// Save original global config
	originalGlobal := Global()
	defer SetGlobal(originalGlobal)

	// Set new global config with temp dir
	newGlobal := DefaultGlobalConfig()
	newGlobal.ConfigDir = tempDir
	SetGlobal(newGlobal)

	// Create directory structure
	// config/osv/azure-linux/azl3/imageconfigs/defaultconfigs/default-raw-x86_64.yml
	osDistDir := filepath.Join(tempDir, "osv", "azure-linux", "azl3")
	defaultConfigDir := filepath.Join(osDistDir, "imageconfigs", "defaultconfigs")
	if err := os.MkdirAll(defaultConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create directory structure: %v", err)
	}

	// Create default config file
	defaultConfigContent := `
image:
  name: default-image
  version: "0.0.1"
target:
  os: azure-linux
  dist: azl3
  arch: x86_64
  imageType: raw
systemConfig:
  name: default-system
  packages:
    - default-pkg
`
	defaultConfigFile := filepath.Join(defaultConfigDir, "default-raw-x86_64.yml")
	if err := os.WriteFile(defaultConfigFile, []byte(defaultConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write default config file: %v", err)
	}

	// Test LoadDefaultConfig
	loader := NewDefaultConfigLoader("azure-linux", "azl3", "x86_64")
	template, err := loader.LoadDefaultConfig("raw")
	if err != nil {
		t.Fatalf("LoadDefaultConfig failed: %v", err)
	}

	if template.Image.Name != "default-image" {
		t.Errorf("Expected image name 'default-image', got '%s'", template.Image.Name)
	}
	if template.SystemConfig.Name != "default-system" {
		t.Errorf("Expected system config name 'default-system', got '%s'", template.SystemConfig.Name)
	}
}

func TestLoadAndMergeTemplate(t *testing.T) {
	// Setup temporary config directory
	tempDir := t.TempDir()

	// Save original global config
	originalGlobal := Global()
	defer SetGlobal(originalGlobal)

	// Set new global config with temp dir
	newGlobal := DefaultGlobalConfig()
	newGlobal.ConfigDir = tempDir
	SetGlobal(newGlobal)

	// Create directory structure for default config
	osDistDir := filepath.Join(tempDir, "osv", "azure-linux", "azl3")
	defaultConfigDir := filepath.Join(osDistDir, "imageconfigs", "defaultconfigs")
	if err := os.MkdirAll(defaultConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create directory structure: %v", err)
	}

	// Create default config file
	defaultConfigContent := `
image:
  name: default-image
  version: "0.0.1"
target:
  os: azure-linux
  dist: azl3
  arch: x86_64
  imageType: raw
systemConfig:
  name: default-system
  packages:
    - default-pkg
`
	defaultConfigFile := filepath.Join(defaultConfigDir, "default-raw-x86_64.yml")
	if err := os.WriteFile(defaultConfigFile, []byte(defaultConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write default config file: %v", err)
	}

	// Create user template file
	userConfigContent := `
image:
  name: user-image
  version: "0.0.2"
target:
  os: azure-linux
  dist: azl3
  arch: x86_64
  imageType: raw
systemConfig:
  name: user-system
  packages:
    - user-pkg
`
	userConfigFile := filepath.Join(tempDir, "user-config.yml")
	if err := os.WriteFile(userConfigFile, []byte(userConfigContent), 0644); err != nil {
		t.Fatalf("Failed to write user config file: %v", err)
	}

	// Test LoadAndMergeTemplate
	template, err := LoadAndMergeTemplate(userConfigFile)
	if err != nil {
		t.Fatalf("LoadAndMergeTemplate failed: %v", err)
	}

	// Verify merged results
	if template.Image.Name != "user-image" {
		t.Errorf("Expected image name 'user-image', got '%s'", template.Image.Name)
	}
	if template.SystemConfig.Name != "user-system" {
		t.Errorf("Expected system config name 'user-system', got '%s'", template.SystemConfig.Name)
	}

	// Verify packages merged (default + user)
	packages := template.GetPackages()
	hasDefault := false
	hasUser := false
	for _, pkg := range packages {
		if pkg == "default-pkg" {
			hasDefault = true
		}
		if pkg == "user-pkg" {
			hasUser = true
		}
	}
	if !hasDefault {
		t.Error("Expected default-pkg in merged packages")
	}
	if !hasUser {
		t.Error("Expected user-pkg in merged packages")
	}
}
