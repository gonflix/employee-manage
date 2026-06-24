package domain

import "time"

const (
	RoleEmployee = "EMPLOYEE"
	RoleAdmin    = "ADMIN"

	AccountStatusActive     = "ACTIVE"
	AccountStatusTerminated = "TERMINATED"
	AccountStatusLocked     = "LOCKED"

	EmploymentStatusEmployed   = "EMPLOYED"
	EmploymentStatusTerminated = "TERMINATED"

	ErrorCodeBlacklistedEmployee = "BLACKLISTED_EMPLOYEE"
)

type Department struct {
	ID          string    `json:"id" gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string    `json:"name" gorm:"column:name;size:100;unique;not null"`
	Description *string   `json:"description,omitempty" gorm:"column:description"`
	CreatedAt   time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (Department) TableName() string {
	return "departments"
}

type Employee struct {
	ID               string      `json:"id" gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	EmployeeNo       string      `json:"employeeNo" gorm:"column:employee_no;size:30;unique;not null"`
	DepartmentID     *string     `json:"departmentId,omitempty" gorm:"column:department_id;type:uuid"`
	Department       *Department `json:"department,omitempty" gorm:"foreignKey:DepartmentID"`
	Name             string      `json:"name" gorm:"column:name;size:100;not null"`
	Email            string      `json:"email" gorm:"column:email;size:255;unique;not null"`
	Phone            *string     `json:"phone,omitempty" gorm:"column:phone;size:30"`
	BirthDate        *time.Time  `json:"birthDate,omitempty" gorm:"column:birth_date;type:date"`
	Address          *string     `json:"address,omitempty" gorm:"column:address"`
	JobTitle         *string     `json:"jobTitle,omitempty" gorm:"column:job_title;size:100"`
	HireDate         time.Time   `json:"hireDate" gorm:"column:hire_date;type:date;not null"`
	TerminationDate  *time.Time  `json:"terminationDate,omitempty" gorm:"column:termination_date;type:date"`
	EmploymentStatus string      `json:"employmentStatus" gorm:"column:employment_status;type:employment_status;default:EMPLOYED;not null"`
	CreatedAt        time.Time   `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt        time.Time   `json:"updatedAt" gorm:"column:updated_at"`
}

func (Employee) TableName() string {
	return "employees"
}

type UserAccount struct {
	ID                string     `json:"id" gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey"`
	EmployeeID        *string    `json:"employeeId,omitempty" gorm:"column:employee_id;type:uuid;unique"`
	Employee          *Employee  `json:"employee,omitempty" gorm:"foreignKey:EmployeeID"`
	LoginID           string     `json:"loginId" gorm:"column:login_id;size:80;unique;not null"`
	PasswordHash      string     `json:"-" gorm:"column:password_hash;not null"`
	Role              string     `json:"role" gorm:"column:role;type:user_role;not null"`
	Status            string     `json:"status" gorm:"column:status;type:account_status;default:ACTIVE;not null"`
	FailedLoginCount  int        `json:"failedLoginCount" gorm:"column:failed_login_count;not null;default:0"`
	LastLoginAt       *time.Time `json:"lastLoginAt,omitempty" gorm:"column:last_login_at"`
	PasswordChangedAt time.Time  `json:"passwordChangedAt" gorm:"column:password_changed_at"`
	CreatedAt         time.Time  `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt         time.Time  `json:"updatedAt" gorm:"column:updated_at"`
}

func (UserAccount) TableName() string {
	return "user_accounts"
}

type TerminatedEmployeeBlacklist struct {
	EmployeeID string    `json:"employeeId" gorm:"column:employee_id;type:uuid;primaryKey"`
	AccountID  *string   `json:"accountId,omitempty" gorm:"column:account_id;type:uuid;unique"`
	LoginID    string    `json:"loginId" gorm:"column:login_id;size:80;not null"`
	Reason     string    `json:"reason" gorm:"column:reason;not null;default:TERMINATED"`
	BlockedAt  time.Time `json:"blockedAt" gorm:"column:blocked_at"`
}

func (TerminatedEmployeeBlacklist) TableName() string {
	return "terminated_employee_blacklist"
}

type LoginRequest struct {
	LoginID  string `json:"loginId"`
	Password string `json:"password"`
}

type AuthTokens struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int64  `json:"expiresIn"`
}

type AuthContext struct {
	AccountID  string
	EmployeeID string
	Role       string
	LoginID    string
}

type UpdateMyProfileRequest struct {
	Phone   *string `json:"phone"`
	Address *string `json:"address"`
}

type CreateEmployeeRequest struct {
	EmployeeNo   string  `json:"employeeNo"`
	LoginID      string  `json:"loginId"`
	Password     string  `json:"password"`
	Name         string  `json:"name"`
	Email        string  `json:"email"`
	Phone        *string `json:"phone"`
	BirthDate    *string `json:"birthDate"`
	Address      *string `json:"address"`
	DepartmentID *string `json:"departmentId"`
	JobTitle     *string `json:"jobTitle"`
	HireDate     string  `json:"hireDate"`
}

type TerminateEmployeeRequest struct {
	TerminationDate string `json:"terminationDate"`
	Reason          string `json:"reason"`
}

type ListEmployeesQuery struct {
	Page    int
	Size    int
	Status  string
	Keyword string
}

type EmployeeList struct {
	Items         []Employee `json:"items"`
	Page          int        `json:"page"`
	Size          int        `json:"size"`
	TotalElements int64      `json:"totalElements"`
	TotalPages    int        `json:"totalPages"`
}

type BackgroundCheck struct {
	CheckedAt                 *time.Time     `json:"checkedAt,omitempty"`
	Status                    string         `json:"status,omitempty"`
	CriminalRecord            *bool          `json:"criminalRecord,omitempty"`
	EmploymentHistoryVerified *bool          `json:"employmentHistoryVerified,omitempty"`
	EducationVerified         *bool          `json:"educationVerified,omitempty"`
	RiskLevel                 string         `json:"riskLevel,omitempty"`
	Raw                       map[string]any `json:"raw,omitempty"`
}

type EmployeeDetail struct {
	Profile         *Employee        `json:"profile"`
	BackgroundCheck *BackgroundCheck `json:"backgroundCheck"`
}

type EmployeeRepository interface {
	FindAccountByLoginID(loginID string) (*UserAccount, error)
	FindAccountByID(accountID string) (*UserAccount, error)
	FindEmployeeByID(employeeID string) (*Employee, error)
	ListEmployees(query ListEmployeesQuery) (*EmployeeList, error)
	CreateEmployeeWithAccount(employee *Employee, account *UserAccount) error
	UpdateEmployeeProfile(employeeID string, phone *string, address *string) (*Employee, error)
	TerminateEmployee(employeeID string, terminationDate time.Time, reason string) (*Employee, error)
	IsEmployeeBlacklisted(employeeID string) (bool, error)
	LoadBlacklistedEmployeeIDs() (map[string]struct{}, error)
	TouchLoginSuccess(accountID string) error
}

type EmployeeService interface {
	LoginEmployee(req LoginRequest) (*AuthTokens, error)
	LoginAdmin(req LoginRequest) (*AuthTokens, error)
	AuthenticateToken(token string) (*AuthContext, error)
	GetMyProfile(ctx AuthContext) (*Employee, error)
	UpdateMyProfile(ctx AuthContext, req UpdateMyProfileRequest) (*Employee, error)
	ListEmployees(ctx AuthContext, query ListEmployeesQuery) (*EmployeeList, error)
	CreateEmployee(ctx AuthContext, req CreateEmployeeRequest) (*Employee, error)
	GetEmployeeDetail(ctx AuthContext, employeeID string) (*EmployeeDetail, error)
	TerminateEmployee(ctx AuthContext, employeeID string, req TerminateEmployeeRequest) (*Employee, error)
}
