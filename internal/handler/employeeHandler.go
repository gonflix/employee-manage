package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gonflix/employee-manage/internal/domain"
	"github.com/labstack/echo/v4"
)

type EmployeeHandler struct {
	employeeService domain.EmployeeService
}

type successResponse struct {
	Timestamp time.Time `json:"timestamp"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Data      any       `json:"data,omitempty"`
}

type errorResponse struct {
	Timestamp time.Time    `json:"timestamp"`
	Code      string       `json:"code,omitempty"`
	Message   string       `json:"message"`
	Errors    []fieldError `json:"errors,omitempty"`
}

type fieldError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func NewEmployeeHandler(employeeService domain.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{employeeService: employeeService}
}

func (h *EmployeeHandler) RegisterRoutes(e *echo.Echo) {
	e.POST("/api/v1/employee/auth/login", h.LoginEmployee)
	e.GET("/api/v1/employee/me", h.GetMyProfile)
	e.PATCH("/api/v1/employee/me", h.UpdateMyProfile)

	e.POST("/api/v1/admin/auth/login", h.LoginAdmin)
	e.GET("/api/v1/admin/employees", h.ListEmployees)
	e.POST("/api/v1/admin/employees", h.CreateEmployee)
	e.GET("/api/v1/admin/employees/:employeeId", h.GetEmployeeDetail)
	e.POST("/api/v1/admin/employees/:employeeId/terminate", h.TerminateEmployee)
}

func (h *EmployeeHandler) LoginEmployee(c echo.Context) error {
	var req domain.LoginRequest
	if err := bind(c, &req); err != nil {
		return writeError(c, err)
	}

	tokens, err := h.employeeService.LoginEmployee(req)
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, http.StatusOK, "OK", "요청이 성공했습니다.", tokens)
}

func (h *EmployeeHandler) LoginAdmin(c echo.Context) error {
	var req domain.LoginRequest
	if err := bind(c, &req); err != nil {
		return writeError(c, err)
	}

	tokens, err := h.employeeService.LoginAdmin(req)
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, http.StatusOK, "OK", "요청이 성공했습니다.", tokens)
}

func (h *EmployeeHandler) GetMyProfile(c echo.Context) error {
	authCtx, err := h.authContext(c)
	if err != nil {
		return writeError(c, err)
	}

	employee, err := h.employeeService.GetMyProfile(*authCtx)
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, http.StatusOK, "OK", "요청이 성공했습니다.", employee)
}

func (h *EmployeeHandler) UpdateMyProfile(c echo.Context) error {
	authCtx, err := h.authContext(c)
	if err != nil {
		return writeError(c, err)
	}

	var req domain.UpdateMyProfileRequest
	if err := bind(c, &req); err != nil {
		return writeError(c, err)
	}

	employee, err := h.employeeService.UpdateMyProfile(*authCtx, req)
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, http.StatusOK, "OK", "요청이 성공했습니다.", employee)
}

func (h *EmployeeHandler) ListEmployees(c echo.Context) error {
	authCtx, err := h.authContext(c)
	if err != nil {
		return writeError(c, err)
	}

	query := domain.ListEmployeesQuery{
		Page:    parsePositiveInt(c.QueryParam("page"), 1),
		Size:    parsePositiveInt(c.QueryParam("size"), 20),
		Status:  c.QueryParam("status"),
		Keyword: c.QueryParam("keyword"),
	}
	employees, err := h.employeeService.ListEmployees(*authCtx, query)
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, http.StatusOK, "OK", "요청이 성공했습니다.", employees)
}

func (h *EmployeeHandler) CreateEmployee(c echo.Context) error {
	authCtx, err := h.authContext(c)
	if err != nil {
		return writeError(c, err)
	}

	var req domain.CreateEmployeeRequest
	if err := bind(c, &req); err != nil {
		return writeError(c, err)
	}

	employee, err := h.employeeService.CreateEmployee(*authCtx, req)
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, http.StatusCreated, "CREATED", "요청이 성공했습니다.", employee)
}

func (h *EmployeeHandler) GetEmployeeDetail(c echo.Context) error {
	authCtx, err := h.authContext(c)
	if err != nil {
		return writeError(c, err)
	}

	detail, err := h.employeeService.GetEmployeeDetail(*authCtx, c.Param("employeeId"))
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, http.StatusOK, "OK", "요청이 성공했습니다.", detail)
}

func (h *EmployeeHandler) TerminateEmployee(c echo.Context) error {
	authCtx, err := h.authContext(c)
	if err != nil {
		return writeError(c, err)
	}

	var req domain.TerminateEmployeeRequest
	if err := bind(c, &req); err != nil {
		return writeError(c, err)
	}

	employee, err := h.employeeService.TerminateEmployee(*authCtx, c.Param("employeeId"), req)
	if err != nil {
		return writeError(c, err)
	}
	return writeSuccess(c, http.StatusOK, "OK", "요청이 성공했습니다.", employee)
}

func (h *EmployeeHandler) authContext(c echo.Context) (*domain.AuthContext, error) {
	header := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return nil, domain.ErrAuthentication
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return nil, domain.ErrAuthentication
	}
	return h.employeeService.AuthenticateToken(token)
}

func bind(c echo.Context, target any) error {
	if err := c.Bind(target); err != nil {
		return domain.ErrInvalidInput
	}
	return nil
}

func writeSuccess(c echo.Context, status int, code string, message string, data any) error {
	return c.JSON(status, successResponse{
		Timestamp: time.Now(),
		Code:      code,
		Message:   message,
		Data:      data,
	})
}

func writeError(c echo.Context, err error) error {
	status, response := mapError(err)
	return c.JSON(status, response)
}

func mapError(err error) (int, errorResponse) {
	response := errorResponse{
		Timestamp: time.Now(),
		Message:   "Internal Server Error",
	}

	switch {
	case errors.Is(err, domain.ErrBlacklistedEmployee):
		response.Code = domain.ErrorCodeBlacklistedEmployee
		response.Message = "퇴사 처리된 직원은 접근할 수 없습니다."
		return http.StatusUnauthorized, response
	case errors.Is(err, domain.ErrAuthentication):
		response.Message = "Authentication Failed"
		return http.StatusUnauthorized, response
	case errors.Is(err, domain.ErrForbidden):
		response.Message = "Forbidden"
		return http.StatusForbidden, response
	case errors.Is(err, domain.ErrInvalidInput):
		response.Message = "Invalid Request"
		return http.StatusBadRequest, response
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, response
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, response
	default:
		return http.StatusInternalServerError, response
	}
}

func parsePositiveInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
