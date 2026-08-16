package server

import (
	"bytes"
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/export"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

// runTemplateByID resolves tmplID, builds the timeline for filter, and
// runs the resolved template's transform pipeline over it. Shared by
// RunTemplate and ExportDocument (when exporting a template run rather
// than an inline document).
func (s *Server) runTemplateByID(
	ctx context.Context,
	tmplID string,
	filter *api.Filter,
	formValues *map[string]any,
) (blocks.Document, error) {
	tmpl, err := s.deps.Store.GetTemplate(ctx, tmplID)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "get template")
	}

	timeline, err := s.queryFullTimeline(ctx, filterToDB(filter))
	if err != nil {
		return nil, err
	}

	form := types.FormValues{}
	if formValues != nil {
		form = *formValues
	}

	doc, err := s.deps.Engine.Run(ctx, tmpl, timeline, form)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "run template")
	}

	return doc, nil
}

// RunTemplate resolves the request's templateId and runs its transform
// pipeline over the (optionally filtered) timeline, returning the
// rendered Document.
//
// RunRequest carries only a templateId (no inline template field in the
// generated schema); unlike ExportDocument, there is no inline-document
// path here.
func (s *Server) RunTemplate(
	ctx context.Context,
	request api.RunTemplateRequestObject,
) (api.RunTemplateResponseObject, error) {
	if request.Body == nil || request.Body.TemplateId == "" {
		status, body := respondError(
			ctx, "run template", validationError("templateId is required"),
		)

		return api.RunTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	doc, err := s.runTemplateByID(
		ctx,
		request.Body.TemplateId,
		request.Body.Filter,
		request.Body.FormValues,
	)
	if err != nil {
		status, body := respondError(ctx, "run template", err)

		return api.RunTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	apiDoc, err := documentToAPI(doc)
	if err != nil {
		status, body := respondError(ctx, "run template", err)

		return api.RunTemplatedefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.RunTemplate200JSONResponse(apiDoc), nil
}

// resolveExportDocument returns the Document ExportDocument should
// serialize: the inline document if the request carries one, else the
// result of running the referenced template over its filter.
func (s *Server) resolveExportDocument(
	ctx context.Context,
	req api.ExportRequest,
) (blocks.Document, error) {
	if req.Document != nil {
		return documentFromAPI(*req.Document)
	}

	if req.TemplateId != nil && *req.TemplateId != "" {
		return s.runTemplateByID(
			ctx, *req.TemplateId, req.Filter, req.FormValues,
		)
	}

	return nil, validationError("document or templateId is required")
}

// exportDocument serializes doc in the requested format.
func exportDocument(
	doc blocks.Document,
	format api.ExportFormat,
) ([]byte, error) {
	var (
		data []byte
		err  error
	)

	switch format {
	case api.Csv:
		data, err = export.ToCSV(doc)
	case api.Pdf:
		data, err = export.ToPDF(doc)
	case api.Json:
		data, err = export.ToJSON(doc)
	default:
		return nil, validationError("unsupported export format")
	}

	if err != nil {
		return nil, ctxerrors.Wrapf(err, "export as %s", format)
	}

	return data, nil
}

// ExportDocument resolves a Document (inline, or by running a template)
// and serializes it in the requested format.
//
// The generated response only carries an application/octet-stream Body +
// ContentLength (VisitExportDocumentResponse hardcodes the content type
// and sets no other headers), so per-format content-type/filename
// distinction isn't expressible within the generated contract — every
// format returns as octet-stream bytes.
func (s *Server) ExportDocument(
	ctx context.Context,
	request api.ExportDocumentRequestObject,
) (api.ExportDocumentResponseObject, error) {
	if request.Body == nil || !request.Body.Format.Valid() {
		status, body := respondError(
			ctx, "export document", validationError("format is invalid"),
		)

		return api.ExportDocumentdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	doc, err := s.resolveExportDocument(ctx, *request.Body)
	if err != nil {
		status, body := respondError(ctx, "export document", err)

		return api.ExportDocumentdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	data, err := exportDocument(doc, request.Body.Format)
	if err != nil {
		status, body := respondError(ctx, "export document", err)

		return api.ExportDocumentdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.ExportDocument200ApplicationoctetStreamResponse{
		Body:          bytes.NewReader(data),
		ContentLength: int64(len(data)),
	}, nil
}
