package pr

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"railgorail/avito/internal/lib/sl"
	"railgorail/avito/internal/repo"
	"railgorail/avito/internal/transport/http/dto"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type prService interface {
	Create(ctx context.Context, prID, prName, authorId string) (*dto.PullRequestSchema, error)
	Merge(ctx context.Context, prID string) (*dto.PullRequestSchema, error)
	Reassign(ctx context.Context, prID, oldRev string) (*dto.ReassignResponse, error)
}

type PrHandler struct {
	log     *slog.Logger
	service prService
}

func NewPrHandler(log *slog.Logger, s prService) *PrHandler {
	return &PrHandler{
		log:     log,
		service: s,
	}
}

type CreateRequest struct {
	PrID     string `json:"pull_request_id"   validate:"required"`
	PrName   string `json:"pull_request_name" validate:"required,min=5"`
	AuthorId string `json:"author_id"         validate:"required"`
}

func (h *PrHandler) Create(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.pr.Create"
	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	ctx := r.Context()

	var input CreateRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		log.Error("failed to decode request body", sl.Err(err))

		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.Error(dto.ErrBadRequest, "bad request"))
		return
	}

	if err := validator.New().Struct(input); err != nil {
		validateError := err.(validator.ValidationErrors)

		log.Error("invalid request", sl.Err(err))

		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.ValidationError(validateError))
		return
	}

	resp, err := h.service.Create(ctx, input.PrID, input.PrName, input.AuthorId)
	if err != nil {
		if errors.Is(err, repo.ErrPRExists) {
			log.Info("pr already exists", sl.Err(err))
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, dto.Error(dto.ErrCodePRExists, err.Error()))
			return
		}
		if errors.Is(err, repo.ErrNotFound) {
			log.Info("resource not found", sl.Err(err))
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, dto.Error(dto.ErrCodeNotFound, err.Error()))
			return
		}
		log.Error("error while creating pr", sl.Err(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.InternalError())
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dto.PrResponse{
		PullRequest: *resp,
	})
}

type MergeRequest struct {
	PrID string `json:"pull_request_id" validate:"required"`
}

func (h *PrHandler) Merge(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.pr.Merge"
	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	ctx := r.Context()

	var input MergeRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		log.Error("failed to decode request body", sl.Err(err))

		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.Error(dto.ErrBadRequest, "bad request"))
		return
	}

	if err := validator.New().Struct(input); err != nil {
		validateError := err.(validator.ValidationErrors)

		log.Error("invalid request", sl.Err(err))

		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.ValidationError(validateError))
		return
	}

	resp, err := h.service.Merge(ctx, input.PrID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			log.Info("pr not found", sl.Err(err))
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, dto.Error(dto.ErrCodeNotFound, err.Error()))
			return
		}
		log.Error("error while merging pr", sl.Err(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.InternalError())
		return
	}

	render.JSON(w, r, dto.PrResponse{PullRequest: *resp})
}

type ReassignRequest struct {
	PrID          string `json:"pull_request_id" validate:"required"`
	OldReviewerID string `json:"old_reviewer_id" validate:"required"`
}

func (h *PrHandler) Reassign(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.pr.Merge"
	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	ctx := r.Context()

	var input ReassignRequest
	if err := render.DecodeJSON(r.Body, &input); err != nil {
		log.Error("failed to decode request body", sl.Err(err))

		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.Error(dto.ErrBadRequest, "bad request"))
		return
	}

	if err := validator.New().Struct(input); err != nil {
		validateError := err.(validator.ValidationErrors)

		log.Error("invalid request", sl.Err(err))

		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.ValidationError(validateError))
		return
	}

	resp, err := h.service.Reassign(ctx, input.PrID, input.OldReviewerID)
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrNotFound):
			log.Info("resource not found", sl.Err(err))
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, dto.Error(dto.ErrCodeNotFound, err.Error()))

		case errors.Is(err, repo.ErrNoCandidate):
			log.Info("no candidate", sl.Err(err))
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, dto.Error(dto.ErrCodeNoCandidate, err.Error()))

		case errors.Is(err, repo.ErrPRMerged):
			log.Info("no candidate", sl.Err(err))
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, dto.Error(dto.ErrCodePRMerged, err.Error()))

		case errors.Is(err, repo.ErrNotAssigned):
			log.Info("no candidate", sl.Err(err))
			render.Status(r, http.StatusConflict)
			render.JSON(w, r, dto.Error(dto.ErrCodeNotAssigned, err.Error()))

		default:
			log.Error("error while reassigning pr", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, dto.InternalError())
		}
		return
	}

	render.JSON(w, r, resp)
}
