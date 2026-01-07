#pragma once
#include <string>

class AuthClient {
public:
    // Проверка прав пользователя на действие
    bool hasPermission(const std::string& userId, const std::string& action) {
        // Заглушка: всегда true
        // В реальном проекте здесь будет запрос к модулю авторизации
        return true;
    }
};
