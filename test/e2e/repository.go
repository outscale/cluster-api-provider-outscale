package e2e

import (
	"context"
	. "github.com/onsi/gomega"
	"os"
	"path/filepath"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/yaml"
)

type clusterctlConfig struct {
	Path   string
	Values map[string]interface{}
}

type providerConfig struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
	Type string `json:"type,omitempty"`
}

func (c *clusterctlConfig) write() {
	data, err := yaml.Marshal(c.Values)
	Expect(err).ToNot(HaveOccurred(), "Failed to convert to yaml the clusterctl config file")
	Expect(os.WriteFile(c.Path, data, 0600)).To(Succeed(), "Failed to write the clusterctl config file")
}

type CreateRepositoryInput struct {
	RepositoryFolder    string
	E2EConfig           *clusterctl.E2EConfig
	FileTransformations []clusterctl.RepositoryFileTransformation
}

func CreateRepository(ctx context.Context, input CreateRepositoryInput) string {
	Expect(input.E2EConfig).ToNot(BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling CreateRepository")
	Expect(os.MkdirAll(input.RepositoryFolder, 0750)).To(Succeed(), "Failed to create the clusterctl local repository folder %s", input.RepositoryFolder)

	providers := []providerConfig{}
	for _, provider := range input.E2EConfig.Providers {
		providerLabel := clusterctlv1.ManifestLabel(provider.Name, clusterctlv1.ProviderType(provider.Type))
		for _, version := range provider.Versions {
			providerURL := filepath.Join(input.RepositoryFolder, providerLabel, version.Name, "components.yaml")
			manifest, err := clusterctl.YAMLForComponentSource(ctx, version)
			Expect(err).ToNot(HaveOccurred(), "Failed to generate the manifest for %q / %q", providerLabel, version.Name)

			sourcePath := filepath.Join(input.RepositoryFolder, providerLabel, version.Name)
			Expect(os.MkdirAll(sourcePath, 0750)).To(Succeed(), "Failed to create the clusterctl local repository folder for %q / %q", providerLabel, version.Name)

			filePath := filepath.Join(sourcePath, "components.yaml")
			Expect(os.WriteFile(filePath, manifest, 0600)).To(Succeed(), "Failed to write manifest in the clusterctl local repository for %q / %q", providerLabel, version.Name)

			destinationPath := filepath.Join(input.RepositoryFolder, providerLabel, version.Name, "components.yaml")
			allFiles := append(provider.Files, version.Files...)
			for _, file := range allFiles {
				data, err := os.ReadFile(file.SourcePath)
				Expect(err).ToNot(HaveOccurred(), "Failed to read file %q / %q", provider.Name, file.SourcePath)

				destinationFile := filepath.Join(filepath.Dir(destinationPath), file.TargetName)
				Expect(os.WriteFile(destinationFile, data, 0600)).To(Succeed(), "Failed to write clusterctl local repository file %q / %q", provider.Name, file.TargetName)
			}
			providers = append(providers, providerConfig{
				Name: provider.Name,
				URL:  providerURL,
				Type: provider.Type,
			})
		}
	}

	overridePath := filepath.Join(input.RepositoryFolder, "overrides")
	Expect(os.MkdirAll(overridePath, 0750)).To(Succeed(), "Failed to create the clusterctl overrides folder %q", overridePath)

	clusterctlConfigFile := &clusterctlConfig{
		Path: filepath.Join(input.RepositoryFolder, "clusterctl-config.yaml"),
		Values: map[string]interface{}{
			"providers":       providers,
			"overridesFolder": overridePath,
		},
	}
	for key := range input.E2EConfig.Variables {
		clusterctlConfigFile.Values[key] = input.E2EConfig.GetVariable(key)
	}
	clusterctlConfigFile.write()

	return clusterctlConfigFile.Path
}
