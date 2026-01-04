-- Таблица пользователей
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255),
    roles JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Таблица refresh токенов
CREATE TABLE refresh_tokens (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    token VARCHAR(512) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Индексы
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

-- Таблица разрешений по ролям (кеширование)
CREATE TABLE role_permissions (
    role VARCHAR(50) PRIMARY KEY,
    permissions JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Добавляем базовые роли
INSERT INTO role_permissions (role, permissions) VALUES
('student', '["user:fullName:write:self", "user:data:read:self", "course:list:read", "course:info:read", "course:testList:read:enrolled", "course:test:read:enrolled"]'),
('teacher', '["user:fullName:write:self", "user:data:read:self", "user:data:read:other", "course:list:read", "course:info:read", "course:info:write:own", "course:testList:read", "course:test:read", "course:test:write:own", "test:create:own", "test:update:own", "test:delete:own", "question:create:own", "question:update:own", "question:delete:own"]'),
('admin', '["user:list:read", "user:fullName:write", "user:data:read", "user:roles:read", "user:roles:write", "user:block:read", "user:block:write", "course:info:write", "course:test:write", "test:create", "test:update", "test:delete", "question:create", "question:update", "question:delete"]');