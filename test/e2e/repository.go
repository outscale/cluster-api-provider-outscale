/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"os"
	"path/filepath"

	gomega "github.com/onsi/gomega"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/yaml"
)

// clusterctlConfig is the config of clusterctl.
type clusterctlConfig struct {
	Path   string
	Values map[string]interface{}
}

// providerConfig is the config of provider.
type providerConfig struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
	Type string `json:"type,omitempty"`
}

// write() write clusterctl config.
func (c *clusterctlConfig) write() {
	data, err := yaml.Marshal(c.Values)
	gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to convert to yaml the clusterctl config file")
	gomega.Expect(os.WriteFile(c.Path, data, 0600)).To(gomega.Succeed(), "Failed to write the clusterctl config file")
}

// CreateRepositoryInput is used with CreateRepository.
type CreateRepositoryInput struct {
	RepositoryFolder    string
	E2EConfig           *clusterctl.E2EConfig
	FileTransformations []clusterctl.RepositoryFileTransformation
}

// CreateRepository is used to create repository.
func CreateRepository(ctx context.Context, input CreateRepositoryInput) string {
	gomega.Expect(input.E2EConfig).ToNot(gomega.BeNil(), "Invalid argument. input.E2EConfig can't be nil when calling CreateRepository")
	gomega.Expect(os.MkdirAll(input.RepositoryFolder, 0750)).To(gomega.Succeed(), "Failed to create the clusterctl local repository folder %s", input.RepositoryFolder)

	providers := []providerConfig{}
	for _, provider := range input.E2EConfig.Providers {
		providerLabel := clusterctlv1.ManifestLabel(provider.Name, clusterctlv1.ProviderType(provider.Type))
		for _, version := range provider.Versions {
			providerURL := filepath.Join(input.RepositoryFolder, providerLabel, version.Name, "components.yaml")
			manifest, err := clusterctl.YAMLForComponentSource(ctx, version)
			gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to generate the manifest for %q / %q", providerLabel, version.Name)

			sourcePath := filepath.Join(input.RepositoryFolder, providerLabel, version.Name)
			gomega.Expect(os.MkdirAll(sourcePath, 0750)).To(gomega.Succeed(), "Failed to create the clusterctl local repository folder for %q / %q", providerLabel, version.Name)

			filePath := filepath.Join(sourcePath, "components.yaml")
			gomega.Expect(os.WriteFile(filePath, manifest, 0600)).To(gomega.Succeed(), "Failed to write manifest in the clusterctl local repository for %q / %q", providerLabel, version.Name)

			destinationPath := filepath.Join(input.RepositoryFolder, providerLabel, version.Name, "components.yaml")
			allFiles := append(provider.Files, version.Files...)
			for _, file := range allFiles {
				data, err := os.ReadFile(file.SourcePath)
				gomega.Expect(err).ToNot(gomega.HaveOccurred(), "Failed to read file %q / %q", provider.Name, file.SourcePath)

				destinationFile := filepath.Join(filepath.Dir(destinationPath), file.TargetName)
				gomega.Expect(os.WriteFile(destinationFile, data, 0600)).To(gomega.Succeed(), "Failed to write clusterctl local repository file %q / %q", provider.Name, file.TargetName)
			}
			providers = append(providers, providerConfig{
				Name: provider.Name,
				URL:  providerURL,
				Type: provider.Type,
			})
		}
	}

	overridePath := filepath.Join(input.RepositoryFolder, "overrides")
	gomega.Expect(os.MkdirAll(overridePath, 0750)).To(gomega.Succeed(), "Failed to create the clusterctl overrides folder %q", overridePath)

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
