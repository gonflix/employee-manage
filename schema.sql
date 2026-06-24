CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE user_role AS ENUM ('EMPLOYEE', 'ADMIN');
CREATE TYPE account_status AS ENUM ('ACTIVE', 'TERMINATED', 'LOCKED');
CREATE TYPE employment_status AS ENUM ('EMPLOYED', 'TERMINATED');

CREATE TABLE departments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_no VARCHAR(30) NOT NULL UNIQUE,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    phone VARCHAR(30),
    birth_date DATE,
    address TEXT,
    job_title VARCHAR(100),
    hire_date DATE NOT NULL,
    termination_date DATE,
    employment_status employment_status NOT NULL DEFAULT 'EMPLOYED',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT employees_termination_date_required
        CHECK (
            (employment_status = 'EMPLOYED' AND termination_date IS NULL)
            OR (employment_status = 'TERMINATED' AND termination_date IS NOT NULL)
        )
);

CREATE TABLE user_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id UUID UNIQUE REFERENCES employees(id) ON DELETE SET NULL,
    login_id VARCHAR(80) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role user_role NOT NULL,
    status account_status NOT NULL DEFAULT 'ACTIVE',
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    last_login_at TIMESTAMPTZ,
    password_changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT employee_account_requires_employee
        CHECK (role = 'ADMIN' OR employee_id IS NOT NULL)
);

CREATE TABLE terminated_employee_blacklist (
    employee_id UUID PRIMARY KEY REFERENCES employees(id) ON DELETE CASCADE,
    account_id UUID UNIQUE REFERENCES user_accounts(id) ON DELETE SET NULL,
    login_id VARCHAR(80) NOT NULL,
    reason TEXT NOT NULL DEFAULT 'TERMINATED',
    blocked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_employees_department_id ON employees(department_id);
CREATE INDEX idx_employees_employment_status ON employees(employment_status);
CREATE INDEX idx_user_accounts_employee_id ON user_accounts(employee_id);
CREATE INDEX idx_user_accounts_role_status ON user_accounts(role, status);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_departments_updated_at
BEFORE UPDATE ON departments
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_employees_updated_at
BEFORE UPDATE ON employees
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER trg_user_accounts_updated_at
BEFORE UPDATE ON user_accounts
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE OR REPLACE FUNCTION block_terminated_employee()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.employment_status = 'TERMINATED' AND OLD.employment_status <> 'TERMINATED' THEN
        UPDATE user_accounts
        SET status = 'TERMINATED'
        WHERE employee_id = NEW.id;

        INSERT INTO terminated_employee_blacklist (employee_id, account_id, login_id)
        SELECT NEW.id, ua.id, ua.login_id
        FROM user_accounts ua
        WHERE ua.employee_id = NEW.id
        ON CONFLICT (employee_id)
        DO UPDATE SET
            account_id = EXCLUDED.account_id,
            login_id = EXCLUDED.login_id,
            blocked_at = now();
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_block_terminated_employee
AFTER UPDATE OF employment_status ON employees
FOR EACH ROW
EXECUTE FUNCTION block_terminated_employee();
