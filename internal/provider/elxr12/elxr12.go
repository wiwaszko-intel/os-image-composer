package elxr12

import (
	"fmt"

	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/config"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/debutils"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/provider"
	"go.uber.org/zap"
)

const (
	baseURL    = "https://deb.debian.org/debian/dists/bookworm/main/"
	configName = "Packages.gz"
	repodata   = ""
)

// repoConfig holds .repo file values
type repoConfig struct {
	Section      string // raw section header
	Name         string // human-readable name from name=
	URL          string
	GPGCheck     bool
	RepoGPGCheck bool
	Enabled      bool
	GPGKey       string
}

// eLxr12 implements provider.Provider
type eLxr12 struct {
	repoURL string
	repoCfg repoConfig
	gzHref  string
	spec    *config.BuildSpec
}

func init() {
	provider.Register(&eLxr12{})
}

// Name returns the unique name of the provider
func (p *eLxr12) Name() string {
	logger := zap.L().Sugar()
	logger.Infof("Name() called - Placeholder: This function will return the provider's unique name.")
	return "eLxr12"
}

// Init will initialize the provider, fetching repo configuration
func (p *eLxr12) Init(spec *config.BuildSpec) error {

	logger := zap.L().Sugar()

	//todo: need to correct of how to get the arch once finalized
	if spec.Arch == "x86_64" {
		spec.Arch = "binary-amd64"
	}
	p.repoURL = baseURL + spec.Arch + "/" + configName

	cfg, err := loadRepoConfig(p.repoURL)
	if err != nil {
		logger.Errorf("parsing repo config failed: %v", err)
		return err
	}

	// logger.Infof("exlr repo URL: %s\n", p.repoURL)
	logger.Infof("exlr repo URL: %s\n", cfg)

	zap.L().Sync() // flush logs if needed
	panic("Stopped by yockgen.")

}

// Packages returns the list of packages
func (p *eLxr12) Packages() ([]provider.PackageInfo, error) {
	logger := zap.L().Sugar()
	logger.Infof("Packages() called - Placeholder: This function will be implemented by the respective owner.")
	return nil, nil
}

// Validate verifies the downloaded files
func (p *eLxr12) Validate(destDir string) error {
	logger := zap.L().Sugar()
	logger.Infof("Validate() called with destDir=%s - Placeholder: This function will be implemented by the respective owner.", destDir)
	return nil
}

// Resolve resolves dependencies
func (p *eLxr12) Resolve(req []provider.PackageInfo, all []provider.PackageInfo) ([]provider.PackageInfo, error) {
	logger := zap.L().Sugar()
	logger.Infof("Resolve() called with destDir=%s - Placeholder: This function will be implemented by the respective owner.")
	return nil, nil
}

// MatchRequested matches requested packages
func (p *eLxr12) MatchRequested(requests []string, all []provider.PackageInfo) ([]provider.PackageInfo, error) {
	logger := zap.L().Sugar()
	logger.Infof("MatchRequested() called - Placeholder: This function will be implemented by the respective owner.")
	return nil, nil
}

func loadRepoConfig(repoUrl string) (repoConfig, error) {
	logger := zap.L().Sugar()

	// Download the debian repo .gz file
	zipFiles, err := debutils.Download(repoUrl)
	if err != nil {
		logger.Errorf("failed to download repo file: %v", err)
		return repoConfig{}, err
	}

	// Decompress the .gz file and store the decompressed file in the same location
	if len(zipFiles) == 0 {
		logger.Errorf("no files downloaded from repo URL: %s", repoUrl)
		return repoConfig{}, fmt.Errorf("no files downloaded from repo URL: %s", repoUrl)
	}
	files, err := debutils.Decompress(zipFiles[0])
	if err != nil {
		logger.Errorf("failed to decompress file: %v", err)
		return repoConfig{}, err
	}
	logger.Infof("decompressed files: %v", files)

	return repoConfig{}, nil
}
