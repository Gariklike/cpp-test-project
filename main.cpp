#include "auth_client.hpp"
#include "main_module.hpp"
#include <iostream>

int main(int argc, char* argv[]) {
    std::string filename = "questions.json";
    if (argc > 1) filename = argv[1];

    std::string userCode = "TEST_CODE"; // код авторизации
    if (argc > 2) userCode = argv[2];

    AuthClient auth;
    std::string token = auth.getAccessToken(userCode);

    if (token.empty()) {
        std::cout << "Не удалось получить токен\n";
        return 1;
    }

    if (!auth.hasPermission(token, "start_test")) {
        std::cout << "У вас нет прав на запуск теста\n";
        return 1;
    }

    MainModule app;
    app.startTest(filename, userCode);

    return 0;
}
