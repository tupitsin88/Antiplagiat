CREATE TABLE works(
    id SERIAL PRIMARY KEY,
    student_name VARCHAR(255) NOT NULL,
    assignment_name VARCHAR(255) NOT NULL,
    file_content BYTEA NOT NULL,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE plagiat_reports (
    id SERIAL PRIMARY KEY,
    work_id INT REFERENCES works(id),
    plagiat_score FLOAT,
    plagiat_sources TEXT,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);