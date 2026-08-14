package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

// ListTemplates returns every saved template, built-in and custom.
func (s *Server) ListTemplates(
	ctx context.Context,
	_ api.ListTemplatesRequestObject,
) (api.ListTemplatesResponseObject, error) {
	tmpls, err := s.deps.Store.ListTemplates(ctx)
	if err != nil {
		status, body := respondError(ctx, "list templates", err)

		return api.ListTemplatesdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	apiTmpls, err := templatesToAPI(tmpls)
	if err != nil {
		status, body := respondError(ctx, "list templates", err)

		return api.ListTemplatesdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.ListTemplates200JSONResponse(apiTmpls), nil
}

// CreateTemplate saves a new, non-builtin template from the client-
// submitted TemplateInput, assigning it a fresh id.
func (s *Server) CreateTemplate(
	ctx context.Context,
	request api.CreateTemplateRequestObject,
) (api.CreateTemplateResponseObject, error) {
	if request.Body == nil || request.Body.Name == "" {
		status, body := respondError(
			ctx, "create template", validationError("name is required"),
		)

		return api.CreateTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	tmpl, err := templateFromInput(uuid.NewString(), *request.Body)
	if err != nil {
		status, body := respondError(ctx, "create template", err)

		return api.CreateTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	if err := s.deps.Store.SaveTemplate(ctx, tmpl); err != nil {
		status, body := respondError(ctx, "create template", err)

		return api.CreateTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	apiTmpl, err := templateToAPI(tmpl)
	if err != nil {
		status, body := respondError(ctx, "create template", err)

		return api.CreateTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.CreateTemplate200JSONResponse(apiTmpl), nil
}

// GenerateTemplate LLM-composes a template draft from a description.
// Nothing is persisted — the caller reviews/edits/saves separately via
// CreateTemplate.
func (s *Server) GenerateTemplate(
	ctx context.Context,
	request api.GenerateTemplateRequestObject,
) (api.GenerateTemplateResponseObject, error) {
	if request.Body == nil || request.Body.Prompt == "" {
		status, body := respondError(
			ctx,
			"generate template",
			validationError("prompt is required"),
		)

		return api.GenerateTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	tmpl, err := s.deps.LLMComposer.GenerateTemplate(
		ctx, request.Body.Prompt,
	)
	if err != nil {
		status, body := respondError(ctx, "generate template", err)

		return api.GenerateTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	apiTmpl, err := templateToAPI(tmpl)
	if err != nil {
		status, body := respondError(ctx, "generate template", err)

		return api.GenerateTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.GenerateTemplate200JSONResponse(apiTmpl), nil
}

// DeleteTemplate removes a custom template. Builtin templates are not
// deletable.
func (s *Server) DeleteTemplate(
	ctx context.Context,
	request api.DeleteTemplateRequestObject,
) (api.DeleteTemplateResponseObject, error) {
	tmpl, err := s.deps.Store.GetTemplate(ctx, request.Id)
	if err != nil {
		status, body := respondError(ctx, "delete template", err)

		return api.DeleteTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	if tmpl.Builtin {
		status, body := respondError(
			ctx,
			"delete template",
			ctxerrors.Wrap(
				commerr.ErrPermissionDenied,
				"builtin templates cannot be deleted",
			),
		)

		return api.DeleteTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	if err := s.deps.Store.DeleteTemplate(ctx, request.Id); err != nil {
		status, body := respondError(ctx, "delete template", err)

		return api.DeleteTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.DeleteTemplate204Response{}, nil
}

// updateTemplateError renders err as the shared default error response for
// the UpdateTemplate operation.
func updateTemplateError(
	ctx context.Context,
	err error,
) api.UpdateTemplateResponseObject {
	status, body := respondError(ctx, "update template", err)

	return api.UpdateTemplatedefaultJSONResponse{
		Body:       body,
		StatusCode: status,
	}
}

// UpdateTemplate replaces a custom template's fields with the submitted
// TemplateInput. Builtin templates are not editable.
func (s *Server) UpdateTemplate(
	ctx context.Context,
	request api.UpdateTemplateRequestObject,
) (api.UpdateTemplateResponseObject, error) {
	existing, err := s.deps.Store.GetTemplate(ctx, request.Id)
	if err != nil {
		return updateTemplateError(ctx, err), nil
	}

	if existing.Builtin {
		return updateTemplateError(
			ctx,
			ctxerrors.Wrap(
				commerr.ErrPermissionDenied,
				"builtin templates cannot be edited",
			),
		), nil
	}

	if request.Body == nil || request.Body.Name == "" {
		return updateTemplateError(
			ctx, validationError("name is required"),
		), nil
	}

	tmpl, err := templateFromInput(existing.ID, *request.Body)
	if err != nil {
		return updateTemplateError(ctx, err), nil
	}

	if err := s.deps.Store.SaveTemplate(ctx, tmpl); err != nil {
		return updateTemplateError(ctx, err), nil
	}

	apiTmpl, err := templateToAPI(tmpl)
	if err != nil {
		return updateTemplateError(ctx, err), nil
	}

	return api.UpdateTemplate200JSONResponse(apiTmpl), nil
}
