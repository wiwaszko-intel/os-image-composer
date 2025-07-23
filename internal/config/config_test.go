package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadYAMLTemplate(t *testing.T) {
	// Create a temporary YAML file with new single object structure
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
  packages:
    - openssh-server
    - docker-ce
    - vim
    - curl
    - wget
  kernel:
    version: "6.12"
    cmdline: "quiet splash"
`

	tmpFile, err := os.CreateTemp("", "test-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading
	template, err := LoadTemplate(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to load YAML template: %v", err)
	}

	// Verify the loaded template
	if template.Target.OS != "azure-linux" {
		t.Errorf("expected OS 'azure-linux', got %s", template.Target.OS)
	}
	if template.Target.Dist != "azl3" {
		t.Errorf("expected dist 'azl3', got %s", template.Target.Dist)
	}
	if template.Target.Arch != "x86_64" {
		t.Errorf("expected arch 'x86_64', got %s", template.Target.Arch)
	}
	if len(template.GetPackages()) != 5 {
		t.Errorf("expected 5 packages, got %d", len(template.GetPackages()))
	}
	if template.Target.ImageType != "raw" {
		t.Errorf("expected imageType 'raw', got %s", template.Target.ImageType)
	}
	if template.GetKernel().Version != "6.12" {
		t.Errorf("expected kernel version '6.12', got %s", template.GetKernel().Version)
	}
}

func TestTemplateHelperMethods(t *testing.T) {
	template := &ImageTemplate{
		Image: ImageInfo{
			Name:    "test-image",
			Version: "1.0.0",
		},
		Target: TargetInfo{
			OS:        "azure-linux",
			Dist:      "azl3",
			Arch:      "x86_64",
			ImageType: "iso",
		},
		// Updated to use single SystemConfig instead of array
		SystemConfig: SystemConfig{
			Name:        "test-config",
			Description: "Test configuration",
			Packages:    []string{"package1", "package2"},
			Kernel: KernelConfig{
				Version: "6.12",
				Cmdline: "quiet",
			},
		},
	}

	// Test provider name mapping
	providerName := template.GetProviderName()
	if providerName != "AzureLinux3" {
		t.Errorf("expected provider 'AzureLinux3', got %s", providerName)
	}

	// Test version mapping
	version := template.GetDistroVersion()
	if version != "3" {
		t.Errorf("expected version '3', got %s", version)
	}

	// Test package extraction
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

	// Test disk config (empty in this test)
	diskConfig := template.GetDiskConfig()
	if diskConfig.Name != "" {
		t.Errorf("expected empty disk config name, got %s", diskConfig.Name)
	}
}

func TestUnsupportedFileFormat(t *testing.T) {
	// Create a temporary file with unsupported extension
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("some content"); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading should fail
	_, err = LoadTemplate(tmpFile.Name())
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
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading
	template, err := LoadTemplate(tmpFile.Name())
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

func TestLoadYAMLTemplateWithUsers(t *testing.T) {
	// Create a temporary YAML file with users configuration under systemConfig
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
  users:
    - name: user
      password: user
      sudo: true
    - name: admin
      passwordHash: "$6$salt$hash"
      groups: ["wheel", "docker"]
      passwordMaxAge: 90
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
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading
	template, err := LoadTemplate(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to load YAML template: %v", err)
	}

	// Verify users configuration
	users := template.GetUsers()
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}

	// Test user by name lookup
	userUser := template.GetUserByName("user")
	if userUser == nil {
		t.Errorf("expected to find user 'user'")
	} else {
		if userUser.Password != "user" {
			t.Errorf("expected user password 'user', got %s", userUser.Password)
		}
		if !userUser.Sudo {
			t.Errorf("expected user to have sudo privileges")
		}
	}

	adminUser := template.GetUserByName("admin")
	if adminUser == nil {
		t.Errorf("expected to find user 'admin'")
	} else {
		if adminUser.PasswordHash != "$6$salt$hash" {
			t.Errorf("expected admin password hash '$6$salt$hash', got %s", adminUser.PasswordHash)
		}
		if len(adminUser.Groups) != 2 {
			t.Errorf("expected admin to have 2 groups, got %d", len(adminUser.Groups))
		}
		if adminUser.PasswordMaxAge != 90 {
			t.Errorf("expected admin passwordMaxAge to be 90, got %d", adminUser.PasswordMaxAge)
		}
	}

	// Test HasUsers method
	if !template.HasUsers() {
		t.Errorf("expected template to have users")
	}
}

func TestUserHelperMethods(t *testing.T) {
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
				{Name: "testuser", Password: "testpass", Sudo: true},
				{Name: "admin", PasswordHash: "$6$test$hash", Groups: []string{"wheel"}, PasswordMaxAge: 365},
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
			{Name: "defaultuser", Password: "defaultpass"},
			{Name: "shared", Password: "defaultshared", Groups: []string{"default"}},
		},
		Packages: []string{"base-package"},
	}

	userConfig := SystemConfig{
		Name: "user",
		Users: []UserConfig{
			{Name: "newuser", Password: "newpass"},
			{Name: "shared", Password: "usershared", Groups: []string{"user", "admin"}, PasswordMaxAge: 180},
		},
		Packages: []string{"user-package"},
	}

	merged := mergeSystemConfig(defaultConfig, userConfig)

	// Test user merge
	if len(merged.Users) != 3 {
		t.Errorf("expected 3 merged users, got %d", len(merged.Users))
	}

	// Test that shared user was properly merged
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
			t.Errorf("expected shared user password to be 'usershared', got %s", sharedUser.Password)
		}
		if len(sharedUser.Groups) != 3 {
			t.Errorf("expected shared user to have 3 groups (merged), got %d", len(sharedUser.Groups))
		}
		if sharedUser.PasswordMaxAge != 180 {
			t.Errorf("expected shared user passwordMaxAge to be 180, got %d", sharedUser.PasswordMaxAge)
		}
	}

	if merged.Name != "user" {
		t.Errorf("expected merged name to be 'user', got %s", merged.Name)
	}
}

func TestUserConfigMerging(t *testing.T) {
	defaultUser := UserConfig{
		Name:           "user",
		Password:       "default",
		Groups:         []string{"users", "default"},
		Home:           "/home/default",
		Shell:          "/bin/bash",
		Sudo:           false,
		PasswordMaxAge: 90,
	}

	userUser := UserConfig{
		Name:           "user",
		Groups:         []string{"admin", "docker"},
		Sudo:           true,
		Shell:          "/bin/zsh",
		PasswordMaxAge: 365,
	}

	merged := mergeUserConfig(defaultUser, userUser)

	// Test that user values override defaults
	if merged.Sudo != true {
		t.Errorf("expected merged sudo to be true, got %t", merged.Sudo)
	}
	if merged.Shell != "/bin/zsh" {
		t.Errorf("expected merged shell to be '/bin/zsh', got %s", merged.Shell)
	}
	if merged.Home != "/home/default" {
		t.Errorf("expected merged home to remain '/home/default', got %s", merged.Home)
	}
	if merged.PasswordMaxAge != 365 {
		t.Errorf("expected merged passwordMaxAge to be 365, got %d", merged.PasswordMaxAge)
	}

	// Test that groups are merged
	expectedGroups := []string{"users", "default", "admin", "docker"}
	if len(merged.Groups) != len(expectedGroups) {
		t.Errorf("expected %d merged groups, got %d", len(expectedGroups), len(merged.Groups))
	}

	// Verify specific groups are present
	groupMap := make(map[string]bool)
	for _, group := range merged.Groups {
		groupMap[group] = true
	}
	for _, expectedGroup := range expectedGroups {
		if !groupMap[expectedGroup] {
			t.Errorf("expected group '%s' to be in merged groups", expectedGroup)
		}
	}
}

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
