package server

import (
	"context"

	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

// ListOwners returns every distinct owner with ingested activity.
func (s *Server) ListOwners(
	ctx context.Context,
	_ api.ListOwnersRequestObject,
) (api.ListOwnersResponseObject, error) {
	owners, err := s.deps.Store.ListOwners(ctx)
	if err != nil {
		status, body := respondError(ctx, "list owners", err)

		return api.ListOwnersdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.ListOwners200JSONResponse(owners), nil
}

// ListRepos returns every distinct repo ingested under the required
// "owner" query parameter.
func (s *Server) ListRepos(
	ctx context.Context,
	request api.ListReposRequestObject,
) (api.ListReposResponseObject, error) {
	if request.Params.Owner == "" {
		status, body := respondError(
			ctx, "list repos", validationError("owner is required"),
		)

		return api.ListReposdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	repos, err := s.deps.Store.ListRepos(ctx, request.Params.Owner)
	if err != nil {
		status, body := respondError(ctx, "list repos", err)

		return api.ListReposdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.ListRepos200JSONResponse(repos), nil
}

// resolvePage converts the 1-indexed, optional API page number into the
// 0-indexed page db.TimelineFilter expects, defaulting/clamping to 1.
func resolvePage(p *int) int {
	page := defaultPage
	if p != nil && *p > 0 {
		page = *p
	}

	return page - 1
}

// resolvePerPage defaults/clamps the optional API per_page into
// [minPerPage, maxPerPage].
func resolvePerPage(p *int) int {
	perPage := defaultPerPage
	if p != nil {
		perPage = *p
	}

	if perPage < minPerPage {
		perPage = minPerPage
	}

	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	return perPage
}

// ListTimeline returns one page of the filtered, newest-first event
// timeline.
func (s *Server) ListTimeline(
	ctx context.Context,
	request api.ListTimelineRequestObject,
) (api.ListTimelineResponseObject, error) {
	params := request.Params

	filter := db.TimelineFilter{
		Page:    resolvePage(params.Page),
		PerPage: resolvePerPage(params.PerPage),
	}
	if params.Owner != nil {
		filter.Owner = *params.Owner
	}

	if params.Repo != nil {
		filter.Repo = *params.Repo
	}

	if params.Type != nil {
		filter.Type = string(*params.Type)
	}

	if params.From != nil {
		filter.From = *params.From
	}

	if params.To != nil {
		filter.To = *params.To
	}

	events, hasMore, err := s.deps.Store.QueryTimeline(ctx, filter)
	if err != nil {
		status, body := respondError(ctx, "list timeline", err)

		return api.ListTimelinedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.ListTimeline200JSONResponse{
		Items:   eventsToAPI(events),
		HasMore: hasMore,
	}, nil
}

// ListSessions derives per-owner work sessions over the filtered range.
//
// The Gap query parameter (a per-request sessionization override) is not
// forwarded: the Sessionizer contract this handler is built against takes
// only a timeline, so gap tuning is the service layer's concern when it
// wires Sessionizer to the sessionize primitive, not this handler's.
func (s *Server) ListSessions(
	ctx context.Context,
	request api.ListSessionsRequestObject,
) (api.ListSessionsResponseObject, error) {
	params := request.Params

	filter := db.TimelineFilter{}
	if params.Owner != nil {
		filter.Owner = *params.Owner
	}

	if params.From != nil {
		filter.From = *params.From
	}

	if params.To != nil {
		filter.To = *params.To
	}

	timeline, err := s.queryFullTimeline(ctx, filter)
	if err != nil {
		status, body := respondError(ctx, "list sessions", err)

		return api.ListSessionsdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	sessions, err := s.deps.Sessionizer.Sessions(ctx, timeline)
	if err != nil {
		status, body := respondError(ctx, "list sessions", err)

		return api.ListSessionsdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.ListSessions200JSONResponse{
		Sessions: sessionsToAPI(sessions),
	}, nil
}
