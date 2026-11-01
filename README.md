# Pulumi provider for Umami

A [Pulumi](https://www.pulumi.com) resource provider for [Umami](https://umami.is), the
privacy-focused web analytics platform. It manages Umami websites through the
[Umami API](https://docs.umami.is/docs/api), so the tracked sites of a self-hosted
instance can live next to the rest of your infrastructure as code.

The plugin is a Go binary released on GitHub. The Node.js SDK is published on npm as
`@midsonlajeanty/pulumi-umami`. Tested against Umami 3.4.1, under the Node.js and Bun runtimes.

## Installation

```sh
npm install @midsonlajeanty/pulumi-umami
```

The SDK declares the plugin download location, so `pulumi up` fetches the matching
plugin from the GitHub release. To install it by hand:

```sh
pulumi plugin install resource umami v0.1.0 --server github://api.github.com/midsonlajeanty/pulumi-umami
```

## Authentication

The provider authenticates with an API key, or with a username and password. An API
key is the recommended option: create one in Umami under Profile, API keys. Password
login is kept for instances that predate API keys. Accounts with two-factor
authentication can only use an API key.

| Config | Environment variable | Role |
|---|---|---|
| `umami:url` | `UMAMI_URL` | Base URL of the Umami server |
| `umami:apiKey` | `UMAMI_API_KEY` | API key, takes precedence over the password |
| `umami:username` | `UMAMI_USERNAME` | Username, with `password` |
| `umami:password` | `UMAMI_PASSWORD` | Password, with `username` |

```sh
pulumi config set umami:url https://analytics.example.com
pulumi config set --secret umami:apiKey umami_xxxxxxxx
```

Credentials are verified when the provider is configured, so a wrong key fails the
preview rather than the first resource.

## Usage

```typescript
import * as umami from "@midsonlajeanty/pulumi-umami";

const site = new umami.Website("docs", {
  name: "Documentation",
  domain: "docs.example.com",
});

export const websiteId = site.id;
```

The resource id is the Umami website id, the value the tracker script needs in its
`data-website-id` attribute.

An explicit provider lets one program manage several Umami instances:

```typescript
const config = new pulumi.Config();

const provider = new umami.Provider("staging", {
  url: "https://analytics.staging.example.com",
  apiKey: config.requireSecret("stagingApiKey"),
});

new umami.Website("staging-docs", { name: "Docs", domain: "docs.staging.example.com" }, { provider });
```

### Website

| Input | Required | Notes |
|---|---|---|
| `name` | yes | Display name |
| `domain` | yes | Domain of the site |
| `teamId` | no | Owning team. Changing it replaces the website, which gives it a new id |

A website deleted in the Umami UI is detected by `pulumi refresh` and recreated by the
next `pulumi up`. An existing website is imported with its id:

```sh
pulumi import umami:index:Website docs 31a48c14-c902-48a4-905b-8c3c0dbb9459
```

## Development

Requirements: Go, the Pulumi CLI, Node.js, Bun, Docker and `make`.

| Target | Role |
|---|---|
| `make build` | Build the plugin into `bin/` |
| `make lint` | gofmt, go vet and golangci-lint |
| `make test` | Unit tests, with a fake Umami server |
| `make schema sdk` | Regenerate `schema.json` and `sdk/nodejs` from the binary |
| `make dist` | Build the release archives for every platform |
| `make e2e` | Run the end-to-end stack under Bun against a local Umami |

The end-to-end test needs an Umami instance on `http://localhost:3999` with the default
`admin` / `umami` account:

```sh
docker compose -f tests/e2e/compose.yaml up -d
make e2e
docker compose -f tests/e2e/compose.yaml down -v
```

Releases are cut by release-please from conventional commits. Merging the release pull
request tags the version, builds the plugin archives and publishes the SDK.

## License

Apache-2.0, see [LICENSE](LICENSE).
