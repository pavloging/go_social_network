CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    author VARCHAR(100) NOT NULL,
    content TEXT NOT NULL,
    tags TEXT[] NOT NULL,                  -- массив строк
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
