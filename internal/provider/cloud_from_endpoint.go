package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
)

// FromEndpoint attempts to load an environment from the given Endpoint.
func FromEndpoint(ctx context.Context, endpoint, environmentName string) (*cloud.Configuration, error) {
	// e.g. https://management.azure.com/ but we need management.azure.com
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, "/")

	uri := fmt.Sprintf("https://%s/metadata/endpoints?api-version=2020-06-01", endpoint)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("retrieving environments from Azure MetaData service: %+v", err)
	}

	var environments []Environment
	if err = json.NewDecoder(resp.Body).Decode(&environments); err != nil {
		return nil, err
	}

	// while the array contains values
	for _, env := range environments {
		if strings.EqualFold(normalizeEnvironmentName(env.Name), environmentName) || (environmentName == "" && len(environments) == 1) {
			// if resourceManager endpoint is empty, assume it's the provided endpoint
			if env.ResourceManager == "" {
				env.ResourceManager = fmt.Sprintf("https://%s/", endpoint)
			}
			return buildAzureCloudConfiguration(env), nil
		}
	}

	return nil, fmt.Errorf("unable to locate metadata for environment %q from custom metadata host %q", environmentName, endpoint)
}

func buildAzureCloudConfiguration(env Environment) *cloud.Configuration {
	return &cloud.Configuration{
		ActiveDirectoryAuthorityHost: env.Authentication.LoginEndpoint,
		Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
			cloud.ResourceManager: {
				Endpoint: env.ResourceManager,
				Audience: env.ResourceManager,
			},
		},
	}
}

func normalizeEnvironmentName(input string) string {
	// Environment is stored as `Azure{Environment}Cloud`
	output := strings.ToLower(input)
	output = strings.TrimPrefix(output, "azure")
	output = strings.TrimSuffix(output, "cloud")

	// however Azure Public is `AzureCloud` in the CLI Profile and not `AzurePublicCloud`.
	if output == "" {
		return "public"
	}
	return output
}

type Environment struct {
	Portal                  string         `json:"portal"`
	Authentication          Authentication `json:"authentication"`
	Media                   string         `json:"media"`
	GraphAudience           string         `json:"graphAudience"`
	Graph                   string         `json:"graph"`
	Name                    string         `json:"name"`
	Suffixes                Suffixes       `json:"suffixes"`
	Batch                   string         `json:"batch"`
	ResourceManager         string         `json:"resourceManager"`
	VmImageAliasDoc         string         `json:"vmImageAliasDoc"`
	ActiveDirectoryDataLake string         `json:"activeDirectoryDataLake"`
	SqlManagement           string         `json:"sqlManagement"`
	Gallery                 string         `json:"gallery"`
}

type Authentication struct {
	LoginEndpoint    string   `json:"loginEndpoint"`
	Audiences        []string `json:"audiences"`
	Tenant           string   `json:"tenant"`
	IdentityProvider string   `json:"identityProvider"`
}

type Suffixes struct {
	AzureDataLakeStoreFileSystem        string `json:"azureDataLakeStoreFileSystem"`
	AcrLoginServer                      string `json:"acrLoginServer"`
	SqlServerHostname                   string `json:"sqlServerHostname"`
	AzureDataLakeAnalyticsCatalogAndJob string `json:"azureDataLakeAnalyticsCatalogAndJob"`
	KeyVaultDns                         string `json:"keyVaultDns"`
	Storage                             string `json:"storage"`
	AzureFrontDoorEndpointSuffix        string `json:"azureFrontDoorEndpointSuffix"`
}
