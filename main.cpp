#include <iostream>
#include <locale>
#include "main_module.hpp"

int main(int argc, char* argv[]) {

    // По умолчанию используем файл questions.json
    std::string filename = "questions.json";
    if (argc > 1) {
        filename = argv[1];
    }

    // Идентификатор пользователя (пока заглушка, позже можно связать с модулем авторизации)
    std::string userId = "user123";

    // Запуск главного модуля
    MainModule app;
    app.startTest(filename, userId);

    return 0;
}
