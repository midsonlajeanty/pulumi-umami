package provider

import (
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
	"github.com/pulumi/pulumi/sdk/v3/go/common/tokens"
)

const Name = "umami"

func New() (p.Provider, error) {
	return infer.NewProviderBuilder().
		WithDisplayName("Umami").
		WithDescription("Manage Umami analytics resources.").
		WithKeywords("pulumi", "umami", "analytics").
		WithHomepage("https://github.com/midsonlajeanty/pulumi-umami").
		WithRepository("https://github.com/midsonlajeanty/pulumi-umami").
		WithPublisher("Louis Midson Lajeanty").
		WithLicense("Apache-2.0").
		WithNamespace("midsonlajeanty").
		WithPluginDownloadURL("github://api.github.com/midsonlajeanty/pulumi-umami").
		WithModuleMap(map[tokens.ModuleName]tokens.ModuleName{"provider": "index"}).
		WithLanguageMap(map[string]any{
			"nodejs": map[string]any{
				"packageName":          "@midsonlajeanty/pulumi-umami",
				"respectSchemaVersion": true,
			},
			"go": map[string]any{
				"importBasePath":       "github.com/midsonlajeanty/pulumi-umami/sdk/go/umami",
				"respectSchemaVersion": true,
			},
		}).
		WithConfig(infer.Config(&Config{})).
		WithResources(infer.Resource(&Website{})).
		Build()
}
