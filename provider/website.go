package provider

import (
	"context"
	"errors"

	"github.com/midsonlajeanty/pulumi-umami/internal/umami"
	p "github.com/pulumi/pulumi-go-provider"
	"github.com/pulumi/pulumi-go-provider/infer"
)

type Website struct{}

type WebsiteArgs struct {
	Name   string  `pulumi:"name"`
	Domain string  `pulumi:"domain"`
	TeamID *string `pulumi:"teamId,optional"`
}

type WebsiteState struct {
	WebsiteArgs
}

func (w *Website) Annotate(a infer.Annotator) {
	a.Describe(w, "A website tracked by Umami. Its resource id is the website id used by the tracker.")
}

func (a *WebsiteArgs) Annotate(annotator infer.Annotator) {
	annotator.Describe(&a.Name, "Display name of the website.")
	annotator.Describe(&a.Domain, "Domain of the website.")
	annotator.Describe(&a.TeamID, "Team that owns the website. Changing it replaces the website.")
}

func client(ctx context.Context) *umami.Client {
	return infer.GetConfig[Config](ctx).client
}

func stateOf(website umami.Website) WebsiteState {
	return WebsiteState{WebsiteArgs{Name: website.Name, Domain: website.Domain, TeamID: website.TeamID}}
}

func (Website) Create(
	ctx context.Context,
	req infer.CreateRequest[WebsiteArgs],
) (infer.CreateResponse[WebsiteState], error) {
	state := WebsiteState{req.Inputs}
	if req.DryRun {
		return infer.CreateResponse[WebsiteState]{Output: state}, nil
	}
	website, err := client(ctx).CreateWebsite(ctx, umami.CreateWebsite{
		Name:   req.Inputs.Name,
		Domain: req.Inputs.Domain,
		TeamID: req.Inputs.TeamID,
	})
	if err != nil {
		return infer.CreateResponse[WebsiteState]{}, err
	}
	return infer.CreateResponse[WebsiteState]{ID: website.ID, Output: state}, nil
}

func (Website) Read(
	ctx context.Context,
	req infer.ReadRequest[WebsiteArgs, WebsiteState],
) (infer.ReadResponse[WebsiteArgs, WebsiteState], error) {
	website, err := client(ctx).GetWebsite(ctx, req.ID)
	if errors.Is(err, umami.ErrNotFound) {
		return infer.ReadResponse[WebsiteArgs, WebsiteState]{}, nil
	}
	if err != nil {
		return infer.ReadResponse[WebsiteArgs, WebsiteState]{}, err
	}
	state := stateOf(website)
	return infer.ReadResponse[WebsiteArgs, WebsiteState]{
		ID:     website.ID,
		Inputs: state.WebsiteArgs,
		State:  state,
	}, nil
}

func (Website) Update(
	ctx context.Context,
	req infer.UpdateRequest[WebsiteArgs, WebsiteState],
) (infer.UpdateResponse[WebsiteState], error) {
	state := WebsiteState{req.Inputs}
	if req.DryRun {
		return infer.UpdateResponse[WebsiteState]{Output: state}, nil
	}
	_, err := client(ctx).UpdateWebsite(ctx, req.ID, umami.UpdateWebsite{
		Name:   req.Inputs.Name,
		Domain: req.Inputs.Domain,
	})
	if err != nil {
		return infer.UpdateResponse[WebsiteState]{}, err
	}
	return infer.UpdateResponse[WebsiteState]{Output: state}, nil
}

func (Website) Delete(
	ctx context.Context,
	req infer.DeleteRequest[WebsiteState],
) (infer.DeleteResponse, error) {
	return infer.DeleteResponse{}, client(ctx).DeleteWebsite(ctx, req.ID)
}

func (Website) Diff(
	_ context.Context,
	req infer.DiffRequest[WebsiteArgs, WebsiteState],
) (p.DiffResponse, error) {
	diff := map[string]p.PropertyDiff{}
	if req.Inputs.Name != req.State.Name {
		diff["name"] = p.PropertyDiff{Kind: p.Update}
	}
	if req.Inputs.Domain != req.State.Domain {
		diff["domain"] = p.PropertyDiff{Kind: p.Update}
	}
	if value(req.Inputs.TeamID) != value(req.State.TeamID) {
		diff["teamId"] = p.PropertyDiff{Kind: p.UpdateReplace}
	}
	return p.DiffResponse{
		HasChanges:          len(diff) > 0,
		DeleteBeforeReplace: true,
		DetailedDiff:        diff,
	}, nil
}
