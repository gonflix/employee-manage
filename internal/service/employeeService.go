package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gonflix/employee-manage/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

const (
	tokenTTLSeconds       = int64(3600)
	defaultRequestTimeout = 5 * time.Second
)

type EmployeeService struct {
	employeeRepository domain.EmployeeRepository
	backgroundAPIURL   string
	httpClient         *http.Client
	tokenSecret        []byte
	blacklistMu        sync.RWMutex
	blacklistedIDs     map[string]struct{}
}

type tokenClaims struct {
	AccountID  string `json:"accountId"`
	EmployeeID string `json:"employeeId,omitempty"`
	Role       string `json:"role"`
	LoginID    string `json:"loginId"`
	jwt.RegisteredClaims
}

func NewEmployeeService(employeeRepository domain.EmployeeRepository) *EmployeeService {
	blacklistedIDs, err := employeeRepository.LoadBlacklistedEmployeeIDs()
	if err != nil {
		blacklistedIDs = map[string]struct{}{}
	}

	return &EmployeeService{
		employeeRepository: employeeRepository,
		backgroundAPIURL:   requiredEnv("BACKGROUND_CHECK_API_URL"),
		httpClient:         &http.Client{Timeout: defaultRequestTimeout},
		tokenSecret:        []byte(requiredEnv("AUTH_SECRET")),
		blacklistedIDs:     blacklistedIDs,
	}
}

func (s *EmployeeService) LoginEmployee(req domain.LoginRequest) (*domain.AuthTokens, error) {
	return s.login(req, domain.RoleEmployee)
}

func (s *EmployeeService) LoginAdmin(req domain.LoginRequest) (*domain.AuthTokens, error) {
	return s.login(req, domain.RoleAdmin)
}

func (s *EmployeeService) AuthenticateToken(token string) (*domain.AuthContext, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return nil, domain.ErrAuthentication
	}
	if claims.ExpiresAt == nil {
		return nil, domain.ErrAuthentication
	}
	if claims.EmployeeID != "" && s.isCachedBlacklisted(claims.EmployeeID) {
		return nil, domain.ErrBlacklistedEmployee
	}

	account, err := s.employeeRepository.FindAccountByID(claims.AccountID)
	if err != nil {
		return nil, domain.ErrAuthentication
	}
	if account.Status != domain.AccountStatusActive {
		if account.EmployeeID != nil && account.Status == domain.AccountStatusTerminated {
			s.addBlacklisted(*account.EmployeeID)
			return nil, domain.ErrBlacklistedEmployee
		}
		return nil, domain.ErrAuthentication
	}
	if account.EmployeeID != nil && s.isBlacklisted(*account.EmployeeID) {
		return nil, domain.ErrBlacklistedEmployee
	}

	employeeID := ""
	if account.EmployeeID != nil {
		employeeID = *account.EmployeeID
	}

	return &domain.AuthContext{
		AccountID:  account.ID,
		EmployeeID: employeeID,
		Role:       account.Role,
		LoginID:    account.LoginID,
	}, nil
}

func (s *EmployeeService) GetMyProfile(ctx domain.AuthContext) (*domain.Employee, error) {
	if ctx.EmployeeID == "" {
		return nil, domain.ErrForbidden
	}
	if s.isBlacklisted(ctx.EmployeeID) {
		return nil, domain.ErrBlacklistedEmployee
	}
	return s.employeeRepository.FindEmployeeByID(ctx.EmployeeID)
}

func (s *EmployeeService) UpdateMyProfile(ctx domain.AuthContext, req domain.UpdateMyProfileRequest) (*domain.Employee, error) {
	if ctx.EmployeeID == "" {
		return nil, domain.ErrForbidden
	}
	if s.isBlacklisted(ctx.EmployeeID) {
		return nil, domain.ErrBlacklistedEmployee
	}
	return s.employeeRepository.UpdateEmployeeProfile(ctx.EmployeeID, req.Phone, req.Address)
}

func (s *EmployeeService) ListEmployees(ctx domain.AuthContext, query domain.ListEmployeesQuery) (*domain.EmployeeList, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if query.Status != "" && query.Status != domain.EmploymentStatusEmployed && query.Status != domain.EmploymentStatusTerminated {
		return nil, domain.ErrInvalidInput
	}
	return s.employeeRepository.ListEmployees(query)
}

func (s *EmployeeService) CreateEmployee(ctx domain.AuthContext, req domain.CreateEmployeeRequest) (*domain.Employee, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.EmployeeNo) == "" ||
		strings.TrimSpace(req.LoginID) == "" ||
		strings.TrimSpace(req.Password) == "" ||
		strings.TrimSpace(req.Name) == "" ||
		strings.TrimSpace(req.Email) == "" ||
		strings.TrimSpace(req.HireDate) == "" {
		return nil, domain.ErrInvalidInput
	}

	hireDate, err := parseDate(req.HireDate)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	var birthDate *time.Time
	if req.BirthDate != nil && strings.TrimSpace(*req.BirthDate) != "" {
		parsedBirthDate, err := parseDate(*req.BirthDate)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		birthDate = &parsedBirthDate
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	employee := &domain.Employee{
		EmployeeNo:       req.EmployeeNo,
		DepartmentID:     req.DepartmentID,
		Name:             req.Name,
		Email:            req.Email,
		Phone:            req.Phone,
		BirthDate:        birthDate,
		Address:          req.Address,
		JobTitle:         req.JobTitle,
		HireDate:         hireDate,
		EmploymentStatus: domain.EmploymentStatusEmployed,
	}
	account := &domain.UserAccount{
		LoginID:           req.LoginID,
		PasswordHash:      string(hash),
		Role:              domain.RoleEmployee,
		Status:            domain.AccountStatusActive,
		PasswordChangedAt: time.Now(),
	}

	if err := s.employeeRepository.CreateEmployeeWithAccount(employee, account); err != nil {
		return nil, err
	}
	return s.employeeRepository.FindEmployeeByID(employee.ID)
}

func (s *EmployeeService) GetEmployeeDetail(ctx domain.AuthContext, employeeID string) (*domain.EmployeeDetail, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	employee, err := s.employeeRepository.FindEmployeeByID(employeeID)
	if err != nil {
		return nil, err
	}
	backgroundCheck, err := s.fetchBackgroundCheck(employee)
	if err != nil {
		return nil, domain.ErrExternalAPI
	}
	return &domain.EmployeeDetail{
		Profile:         employee,
		BackgroundCheck: backgroundCheck,
	}, nil
}

func (s *EmployeeService) TerminateEmployee(ctx domain.AuthContext, employeeID string, req domain.TerminateEmployeeRequest) (*domain.Employee, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.TerminationDate) == "" {
		return nil, domain.ErrInvalidInput
	}

	terminationDate, err := parseDate(req.TerminationDate)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = domain.EmploymentStatusTerminated
	}

	employee, err := s.employeeRepository.TerminateEmployee(employeeID, terminationDate, reason)
	if err != nil {
		return nil, err
	}
	s.addBlacklisted(employeeID)
	return employee, nil
}

func (s *EmployeeService) login(req domain.LoginRequest, role string) (*domain.AuthTokens, error) {
	if strings.TrimSpace(req.LoginID) == "" || strings.TrimSpace(req.Password) == "" {
		return nil, domain.ErrAuthentication
	}

	account, err := s.employeeRepository.FindAccountByLoginID(req.LoginID)
	if err != nil {
		return nil, domain.ErrAuthentication
	}
	if account.Role != role {
		if role == domain.RoleEmployee {
			return nil, domain.ErrAuthentication
		}
		return nil, domain.ErrForbidden
	}
	if account.EmployeeID != nil && s.isBlacklisted(*account.EmployeeID) {
		return nil, domain.ErrBlacklistedEmployee
	}
	if account.Status == domain.AccountStatusTerminated {
		if account.EmployeeID != nil {
			s.addBlacklisted(*account.EmployeeID)
		}
		return nil, domain.ErrBlacklistedEmployee
	}
	if account.Status != domain.AccountStatusActive {
		return nil, domain.ErrAuthentication
	}
	if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(req.Password)) != nil {
		return nil, domain.ErrAuthentication
	}
	if err := s.employeeRepository.TouchLoginSuccess(account.ID); err != nil {
		return nil, err
	}

	employeeID := ""
	if account.EmployeeID != nil {
		employeeID = *account.EmployeeID
	}

	token, err := s.signToken(tokenClaims{
		AccountID:  account.ID,
		EmployeeID: employeeID,
		Role:       account.Role,
		LoginID:    account.LoginID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(tokenTTLSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	})
	if err != nil {
		return nil, err
	}

	return &domain.AuthTokens{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   tokenTTLSeconds,
	}, nil
}

func (s *EmployeeService) fetchBackgroundCheck(employee *domain.Employee) (*domain.BackgroundCheck, error) {
	payload, _ := json.Marshal(map[string]any{
		"employeeId": employee.ID,
		"name":       employee.Name,
		"email":      employee.Email,
	})

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		reqCtx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, s.backgroundAPIURL, bytes.NewReader(payload))
		if err != nil {
			cancel()
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.httpClient.Do(req)
		if err == nil && resp != nil {
			body, readErr := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			cancel()
			if readErr != nil {
				return nil, readErr
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return parseBackgroundCheck(body), nil
			}

			lastErr = domain.ErrExternalAPI
			if attempt < 3 {
				time.Sleep(retryDelay(attempt, resp.StatusCode, body))
				continue
			}
			return nil, lastErr
		}
		cancel()

		lastErr = err
		if attempt < 3 {
			time.Sleep(retryDelay(attempt, 0, nil))
		}
	}

	return nil, lastErr
}

func parseBackgroundCheck(body []byte) *domain.BackgroundCheck {
	now := time.Now()
	raw := map[string]any{}
	_ = json.Unmarshal(body, &raw)

	check := &domain.BackgroundCheck{
		CheckedAt: &now,
		Raw:       raw,
	}
	if status, ok := raw["status"].(string); ok {
		check.Status = status
	}
	if riskLevel, ok := raw["riskLevel"].(string); ok {
		check.RiskLevel = riskLevel
	}
	if criminalRecord, ok := raw["criminalRecord"].(bool); ok {
		check.CriminalRecord = &criminalRecord
	}
	if employmentHistoryVerified, ok := raw["employmentHistoryVerified"].(bool); ok {
		check.EmploymentHistoryVerified = &employmentHistoryVerified
	}
	if educationVerified, ok := raw["educationVerified"].(bool); ok {
		check.EducationVerified = &educationVerified
	}
	return check
}

func retryDelay(attempt int, statusCode int, body []byte) time.Duration {
	// 503 Service Unavailable with Retry-After header
	if statusCode == http.StatusServiceUnavailable {
		var payload struct {
			RetryAfter int `json:"retryAfter"`
		}
		if json.Unmarshal(body, &payload) == nil && payload.RetryAfter > 0 {
			return time.Duration(payload.RetryAfter) * time.Second
		}
	}

	// Exponential backoff with jitter
	base := time.Duration(math.Pow(2, float64(attempt-1)) * float64(200*time.Millisecond))
	jitter := time.Duration(randomInt63(int64(100 * time.Millisecond)))
	return base + jitter
}

func randomInt63(max int64) int64 {
	if max <= 0 {
		return 0
	}
	value, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0
	}
	return value.Int64()
}

func (s *EmployeeService) signToken(claims tokenClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.tokenSecret)
}

func (s *EmployeeService) parseToken(token string) (*tokenClaims, error) {
	claims := &tokenClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(parsedToken *jwt.Token) (any, error) {
		if parsedToken.Method != jwt.SigningMethodHS256 {
			return nil, domain.ErrAuthentication
		}
		return s.tokenSecret, nil
	})
	if err != nil || parsedToken == nil || !parsedToken.Valid {
		return nil, domain.ErrAuthentication
	}
	if claims.AccountID == "" || claims.Role == "" {
		return nil, domain.ErrAuthentication
	}
	return claims, nil
}

func (s *EmployeeService) isBlacklisted(employeeID string) bool {
	if s.isCachedBlacklisted(employeeID) {
		return true
	}
	blacklisted, err := s.employeeRepository.IsEmployeeBlacklisted(employeeID)
	if err != nil {
		return false
	}
	if blacklisted {
		s.addBlacklisted(employeeID)
	}
	return blacklisted
}

func (s *EmployeeService) isCachedBlacklisted(employeeID string) bool {
	s.blacklistMu.RLock()
	defer s.blacklistMu.RUnlock()
	_, ok := s.blacklistedIDs[employeeID]
	return ok
}

func (s *EmployeeService) addBlacklisted(employeeID string) {
	if employeeID == "" {
		return
	}
	s.blacklistMu.Lock()
	defer s.blacklistMu.Unlock()
	s.blacklistedIDs[employeeID] = struct{}{}
}

func requireAdmin(ctx domain.AuthContext) error {
	if ctx.Role != domain.RoleAdmin {
		return domain.ErrForbidden
	}
	return nil
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", value)
}

func requiredEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("%s environment variable is required", key)
	}
	return value
}
