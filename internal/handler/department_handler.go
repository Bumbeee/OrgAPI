package handler

import (
	"OrgAPI/internal/model"
	"OrgAPI/internal/repository"
	"OrgAPI/internal/service"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

type DepartmentService interface {
	CreateDepartment(ctx context.Context, dto service.CreateDepartmentDTO) (*model.Department, error)
	GetByID(ctx context.Context, dto service.GetDepartmentDTO) (*model.Department, error)
	UpdateParent(ctx context.Context, dto service.UpdateParentDTO) (*model.Department, error)
	Delete(ctx context.Context, dto service.DeleteDepartmentDTO) error
}

type DepartmentHandler struct {
	service DepartmentService
	log     *slog.Logger
}

func NewDepartmentHandler(service DepartmentService, log *slog.Logger) *DepartmentHandler {
	return &DepartmentHandler{service: service, log: log}
}

func (h *DepartmentHandler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	var dto service.CreateDepartmentDTO
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid input", err)
		return
	}

	department, err := h.service.CreateDepartment(r.Context(), dto)
	if err != nil {
		if errors.Is(err, service.ErrInvalidParams) {
			h.writeError(r.Context(), w, http.StatusBadRequest, "validation failed", err)
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			h.writeError(r.Context(), w, http.StatusNotFound, "parent department not found", err)
			return
		}
		if errors.Is(err, repository.ErrAlreadyExists) {
			h.writeError(r.Context(), w, http.StatusConflict, "department name already exists in parent", err)
			return
		}
		h.log.Error("department creation failed", "error", err)
		h.writeError(r.Context(), w, http.StatusInternalServerError, "internal error", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/departments")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(department); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

func (h *DepartmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid department id", err)
		return
	}

	query := r.URL.Query()
	depth, err := parseOptionalInt(query.Get("depth"), 1)
	if err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid depth", err)
		return
	}

	includeEmployees, err := strconv.ParseBool(query.Get("include_employees"))
	if query.Get("include_employees") == "" {
		includeEmployees = true
		err = nil
	}
	if err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid include_employees", err)
		return
	}

	dto := service.GetDepartmentDTO{
		ID: id,
		Opts: repository.GetDepartmentOptions{
			Depth:            depth,
			IncludeEmployees: includeEmployees,
			SortBy:           query.Get("sort_by"),
		},
	}

	department, err := h.service.GetByID(r.Context(), dto)
	if err != nil {
		if errors.Is(err, service.ErrInvalidParams) {
			h.writeError(r.Context(), w, http.StatusBadRequest, "validation failed", err)
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			h.writeError(r.Context(), w, http.StatusNotFound, "department not found", err)
			return
		}
		h.writeError(r.Context(), w, http.StatusInternalServerError, "internal error", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(department); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

func (h *DepartmentHandler) UpdateParent(w http.ResponseWriter, r *http.Request) {
	var req updateDepartmentRequest
	defer r.Body.Close()

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid department id", err)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid json", err)
		return
	}

	dto := service.UpdateParentDTO{
		ID:   id,
		Name: req.Name,
	}
	if len(req.ParentID) > 0 {
		dto.ParentIDSet = true
		if string(req.ParentID) != "null" {
			var parentID int
			if err := json.Unmarshal(req.ParentID, &parentID); err != nil {
				h.writeError(r.Context(), w, http.StatusBadRequest, "invalid parent_id", err)
				return
			}
			dto.ParentID = &parentID
		}
	}

	department, err := h.service.UpdateParent(r.Context(), dto)
	if err != nil {
		if errors.Is(err, service.ErrInvalidParams) || errors.Is(err, service.ErrInvalidID) || errors.Is(err, service.ErrSelfParent) {
			h.writeError(r.Context(), w, http.StatusBadRequest, "validation failed", err)
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			h.writeError(r.Context(), w, http.StatusNotFound, "department not found", err)
			return
		}
		if errors.Is(err, repository.ErrAlreadyExists) {
			h.writeError(r.Context(), w, http.StatusConflict, "department name already exists in parent", err)
			return
		}
		if errors.Is(err, repository.ErrCycle) || errors.Is(err, service.ErrCyclicDependency) {
			h.writeError(r.Context(), w, http.StatusConflict, "department tree cycle", err)
			return
		}
		h.writeError(r.Context(), w, http.StatusInternalServerError, "internal error", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/departments/"+strconv.Itoa(department.ID))
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(department); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

func (h *DepartmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid department id", err)
		return
	}

	query := r.URL.Query()
	reassignToID, err := parseOptionalInt(query.Get("reassign_to_department_id"), 0)
	if err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid reassign_to_department_id", err)
		return
	}
	mode := query.Get("mode")
	if mode == "" {
		mode = service.DeleteModeCascade
	}

	dto := service.DeleteDepartmentDTO{
		ID: id,
		Opts: repository.DeleteOptions{
			Mode:         mode,
			ReassingToID: reassignToID,
		},
	}

	if err := h.service.Delete(r.Context(), dto); err != nil {
		if errors.Is(err, service.ErrInvalidParams) {
			h.writeError(r.Context(), w, http.StatusBadRequest, "validation failed", err)
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			h.writeError(r.Context(), w, http.StatusNotFound, "department not found", err)
			return
		}
		if errors.Is(err, repository.ErrHasChildren) {
			h.writeError(r.Context(), w, http.StatusConflict, "department has children", err)
			return
		}
		if errors.Is(err, repository.ErrAlreadyExists) {
			h.writeError(r.Context(), w, http.StatusConflict, "child department name already exists in target parent", err)
			return
		}
		if errors.Is(err, repository.ErrCycle) {
			h.writeError(r.Context(), w, http.StatusConflict, "invalid reassignment target", err)
			return
		}
		h.writeError(r.Context(), w, http.StatusInternalServerError, "internal error", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *DepartmentHandler) writeError(ctx context.Context, w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{Error: message}
	if err != nil && h.log.Enabled(ctx, slog.LevelDebug) {
		response.Details = err.Error()
	}

	_ = json.NewEncoder(w).Encode(response)
}

func parseOptionalInt(value string, defaultValue int) (int, error) {
	if value == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(value)
}

type updateDepartmentRequest struct {
	Name     *string         `json:"name"`
	ParentID json.RawMessage `json:"parent_id"`
}
