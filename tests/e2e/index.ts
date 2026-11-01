import * as pulumi from "@pulumi/pulumi";
import * as umami from "@midsonlajeanty/pulumi-umami";

const config = new pulumi.Config();
const siteName = config.get("siteName") ?? "Example";

const site = new umami.Website("example", {
  name: siteName,
  domain: "example.com",
});

export const websiteId = site.id;
export const websiteName = site.name;
