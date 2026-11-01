package provider

import (
	"context"
	"errors"

	"github.com/midsonlajeanty/pulumi-umami/internal/umami"
	"github.com/pulumi/pulumi-go-provider/infer"
)

type Config struct {
	URL      *string `pulumi:"url,optional"`
	APIKey   *string `pulumi:"apiKey,optional" provider:"secret"`
	Username *string `pulumi:"username,optional"`
	Password *string `pulumi:"password,optional" provider:"secret"`

	client *umami.Client
}

func (c *Config) Annotate(a infer.Annotator) {
	a.Describe(&c.URL, "Base URL of the Umami server, for example https://analytics.example.com.")
	a.Describe(&c.APIKey, "Umami API key. Takes precedence over username and password.")
	a.Describe(&c.Username, "Username, used with password when no API key is set.")
	a.Describe(&c.Password, "Password, used with username when no API key is set.")
	a.SetDefault(&c.URL, nil, "UMAMI_URL")
	a.SetDefault(&c.APIKey, nil, "UMAMI_API_KEY")
	a.SetDefault(&c.Username, nil, "UMAMI_USERNAME")
	a.SetDefault(&c.Password, nil, "UMAMI_PASSWORD")
}

func (c *Config) Configure(ctx context.Context) error {
	if value(c.URL) == "" {
		return errors.New("umami: url is required, set the url config or UMAMI_URL")
	}
	client, err := umami.NewClient(value(c.URL), umami.Credentials{
		APIKey:   value(c.APIKey),
		Username: value(c.Username),
		Password: value(c.Password),
	})
	if err != nil {
		return err
	}
	if err := client.Verify(ctx); err != nil {
		return err
	}
	c.client = client
	return nil
}

func value(pointer *string) string {
	if pointer == nil {
		return ""
	}
	return *pointer
}
