package handler

import (
	"OrgAPI/internal/model"
	"OrgAPI/internal/repository"
	"OrgAPI/internal/service"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type EmployeeService interface {
	CreateEmployee(ctx context.Context, dto service.CreateEmployeeDTO) (*model.Employee, error)
}

type EmployeeHandler struct {
	service EmployeeService
	log     *slog.Logger
}

func NewEmployeeHandler(service EmployeeService, log *slog.Logger) *EmployeeHandler {
	return &EmployeeHandler{service: service, log: log}
}

func (h *EmployeeHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	var dto service.CreateEmployeeDTO

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid json", err)
		return
	}

	id, err := departmentIDFromPath(r.URL.Path)
	if err != nil {
		h.writeError(r.Context(), w, http.StatusBadRequest, "invalid department id", err)
		return
	}
	dto.DepartmentID = id

	employee, err := h.service.CreateEmployee(r.Context(), dto)
	if err != nil {
		if errors.Is(err, service.ErrInvalidParams) || errors.Is(err, service.ErrInvalidID) || errors.Is(err, service.ErrEmployeeValidate) {
			h.writeError(r.Context(), w, http.StatusBadRequest, "validation failed", err)
			return
		}
		if errors.Is(err, repository.ErrNotFound) {
			h.writeError(r.Context(), w, http.StatusNotFound, "department not found", err)
			return
		}
		h.log.Error("employee creation failed", "error", err)
		h.writeError(r.Context(), w, http.StatusInternalServerError, "internal error", nil)
		return
	}

	location := fmt.Sprintf("/departments/%d/employees/%d", employee.DepartmentID, employee.ID)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(employee); err != nil {
		h.log.Error("failed to encode response", "error", err)
	}
}

func (h *EmployeeHandler) writeError(ctx context.Context, w http.ResponseWriter, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := ErrorResponse{Error: message}
	if err != nil && h.log.Enabled(ctx, slog.LevelDebug) {
		response.Details = err.Error()
	}

	_ = json.NewEncoder(w).Encode(response)
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

func departmentIDFromPath(path string) (int, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "departments" || parts[2] != "employees" {
		return 0, service.ErrInvalidID
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil || id <= 0 {
		return 0, service.ErrInvalidID
	}
	return id, nil
}
