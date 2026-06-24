package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/gonflix/employee-manage/internal/domain"
	"gorm.io/gorm"
)

type EmployeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

func (r *EmployeeRepository) FindAccountByLoginID(loginID string) (*domain.UserAccount, error) {
	var account domain.UserAccount
	err := r.db.Preload("Employee.Department").
		Where("login_id = ?", loginID).
		First(&account).Error
	return accountResult(&account, err)
}

func (r *EmployeeRepository) FindAccountByID(accountID string) (*domain.UserAccount, error) {
	var account domain.UserAccount
	err := r.db.Preload("Employee.Department").
		Where("id = ?", accountID).
		First(&account).Error
	return accountResult(&account, err)
}

func (r *EmployeeRepository) FindEmployeeByID(employeeID string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.Preload("Department").
		Where("id = ?", employeeID).
		First(&employee).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	return &employee, err
}

func (r *EmployeeRepository) ListEmployees(query domain.ListEmployeesQuery) (*domain.EmployeeList, error) {
	page := max(query.Page, 1)
	size := min(max(query.Size, 20), 100)

	db := r.db.Model(&domain.Employee{}).Preload("Department")
	if query.Status != "" {
		db = db.Where("employment_status = ?", query.Status)
	}
	if query.Keyword != "" {
		keyword := "%" + strings.ToLower(query.Keyword) + "%"
		db = db.Where(
			"LOWER(name) LIKE ? OR LOWER(employee_no) LIKE ? OR LOWER(email) LIKE ?",
			keyword,
			keyword,
			keyword,
		)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var employees []domain.Employee
	if err := db.Order("created_at DESC").
		Limit(size).
		Offset((page - 1) * size).
		Find(&employees).Error; err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(size) - 1) / int64(size))
	}

	return &domain.EmployeeList{
		Items:         employees,
		Page:          page,
		Size:          size,
		TotalElements: total,
		TotalPages:    totalPages,
	}, nil
}

func (r *EmployeeRepository) CreateEmployeeWithAccount(employee *domain.Employee, account *domain.UserAccount) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(employee).Error; err != nil {
			return mapDuplicateError(err)
		}

		account.EmployeeID = &employee.ID
		if err := tx.Create(account).Error; err != nil {
			return mapDuplicateError(err)
		}

		return nil
	})
}

func (r *EmployeeRepository) UpdateEmployeeProfile(employeeID string, phone *string, address *string) (*domain.Employee, error) {
	updates := map[string]any{}
	if phone != nil {
		updates["phone"] = phone
	}
	if address != nil {
		updates["address"] = address
	}
	if len(updates) == 0 {
		return r.FindEmployeeByID(employeeID)
	}

	result := r.db.Model(&domain.Employee{}).
		Where("id = ? AND employment_status = ?", employeeID, domain.EmploymentStatusEmployed).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.ErrNotFound
	}

	return r.FindEmployeeByID(employeeID)
}

func (r *EmployeeRepository) TerminateEmployee(employeeID string, terminationDate time.Time, reason string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", employeeID).First(&employee).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}
		if employee.EmploymentStatus == domain.EmploymentStatusTerminated {
			return domain.ErrInvalidInput
		}

		if err := tx.Model(&domain.Employee{}).
			Where("id = ?", employeeID).
			Updates(map[string]any{
				"employment_status": domain.EmploymentStatusTerminated,
				"termination_date":  terminationDate,
			}).Error; err != nil {
			return err
		}

		var account domain.UserAccount
		accountErr := tx.Where("employee_id = ?", employeeID).First(&account).Error
		if accountErr != nil && !errors.Is(accountErr, gorm.ErrRecordNotFound) {
			return accountErr
		}
		if accountErr == nil {
			if err := tx.Model(&domain.UserAccount{}).
				Where("id = ?", account.ID).
				Update("status", domain.AccountStatusTerminated).Error; err != nil {
				return err
			}

			blacklist := domain.TerminatedEmployeeBlacklist{
				EmployeeID: employeeID,
				AccountID:  &account.ID,
				LoginID:    account.LoginID,
				Reason:     reason,
				BlockedAt:  time.Now(),
			}
			if err := tx.Save(&blacklist).Error; err != nil {
				return err
			}
		}

		return tx.Preload("Department").Where("id = ?", employeeID).First(&employee).Error
	})
	if err != nil {
		return nil, err
	}
	return &employee, nil
}

func (r *EmployeeRepository) IsEmployeeBlacklisted(employeeID string) (bool, error) {
	var count int64
	err := r.db.Model(&domain.TerminatedEmployeeBlacklist{}).
		Where("employee_id = ?", employeeID).
		Count(&count).Error
	return count > 0, err
}

func (r *EmployeeRepository) LoadBlacklistedEmployeeIDs() (map[string]struct{}, error) {
	var rows []domain.TerminatedEmployeeBlacklist
	if err := r.db.Find(&rows).Error; err != nil {
		return nil, err
	}

	ids := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		ids[row.EmployeeID] = struct{}{}
	}
	return ids, nil
}

func (r *EmployeeRepository) TouchLoginSuccess(accountID string) error {
	now := time.Now()
	return r.db.Model(&domain.UserAccount{}).
		Where("id = ?", accountID).
		Updates(map[string]any{
			"failed_login_count": 0,
			"last_login_at":      now,
		}).Error
}

func accountResult(account *domain.UserAccount, err error) (*domain.UserAccount, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	return account, err
}

func mapDuplicateError(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return domain.ErrConflict
	}
	return err
}
