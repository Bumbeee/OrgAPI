-- +goose Up
-- +goose StatementBegin
CREATE TABLE departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    parent_id INT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT departments_name_not_blank CHECK (length(btrim(name)) BETWEEN 1 AND 200)
);

ALTER TABLE departments
    ADD CONSTRAINT fk_department_parent
    FOREIGN KEY (parent_id) REFERENCES departments(id)
    ON DELETE CASCADE;

CREATE UNIQUE INDEX idx_departments_parent_name
    ON departments(parent_id, name)
    WHERE parent_id IS NOT NULL;

CREATE UNIQUE INDEX idx_departments_root_name
    ON departments(name)
    WHERE parent_id IS NULL;

CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    department_id INT NOT NULL,
    full_name VARCHAR(200) NOT NULL,
    position VARCHAR(200) NOT NULL,
    hired_at DATE NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT employees_full_name_not_blank CHECK (length(btrim(full_name)) BETWEEN 1 AND 200),
    CONSTRAINT employees_position_not_blank CHECK (length(btrim(position)) BETWEEN 1 AND 200)
);

ALTER TABLE employees
    ADD CONSTRAINT fk_employee_department
    FOREIGN KEY (department_id) REFERENCES departments(id)
    ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS departments;
-- +goose StatementEnd
