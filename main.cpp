#include "main_module.hpp"

int main(int argc, char* argv[]) {
    setlocale(LC_ALL, "Russian");
    std::string filename = "questions.json";
    if (argc > 1) filename = argv[1];

    MainModule app;
    app.startTest(filename);

    return 0;
}
