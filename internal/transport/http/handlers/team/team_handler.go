package team

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

type teamService interface {
	Add(ctx context.Context, teamName string, users []dto.TeamMember) (*dto.TeamSchema, error)
	Get(ctx context.Context, teamName string) (*dto.TeamSchema, error)
}

type TeamHandler struct {
	log     *slog.Logger
	service teamService
}

func NewTeamHandler(log *slog.Logger, s teamService) *TeamHandler {
	return &TeamHandler{
		log:     log,
		service: s,
	}
}

type TeamAddRequest struct {
	TeamName string           `json:"team_name" validate:"required"`
	Members  []dto.TeamMember `json:"members"   validate:"required,dive"`
}

func (h *TeamHandler) Add(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.team.Add"
	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	ctx := r.Context()

	var input TeamAddRequest

	if err := render.DecodeJSON(r.Body, &input); err != nil {
		log.Error("failed to decode request body", sl.Err(err))

		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.Error(dto.ErrBadRequest, "bad request"))
		return
	}

	if err := validator.New().Struct(input); err != nil {
		validateError := func() validator.ValidationErrors {
			var target validator.ValidationErrors
			_ = errors.As(err, &target)
			return target
		}()

		log.Error("invalid request", sl.Err(err))

		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.ValidationError(validateError))
		return
	}

	resp, err := h.service.Add(ctx, input.TeamName, input.Members)
	if err != nil {
		if errors.Is(err, repo.ErrTeamExists) {
			log.Error("team exists", sl.Err(err))

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, dto.Error(dto.ErrCodeTeamExists, err.Error()))
			return
		}
		log.Error("error while saving team", sl.Err(err))

		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.InternalError())
		return
	}

	log.Info("team created successfully")
	render.Status(r, http.StatusCreated)
	render.JSON(w, r, dto.TeamResponse{Team: *resp})
}

func (h *TeamHandler) Get(w http.ResponseWriter, r *http.Request) {
	const op = "handlers.team.Get"
	log := h.log.With(
		slog.String("op", op),
		slog.String("request_id", middleware.GetReqID(r.Context())),
	)

	ctx := r.Context()

	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, dto.Error(dto.ErrBadRequest, "team_name is required"))
		return
	}

	resp, err := h.service.Get(ctx, teamName)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			log.Info("team not found", sl.Err(err))
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, dto.Error(dto.ErrCodeNotFound, err.Error()))
			return
		}
		log.Error("error while retrieving team", sl.Err(err))
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, dto.InternalError())
		return
	}
	log.Info("team retrieved")
	render.JSON(w, r, resp)
}
