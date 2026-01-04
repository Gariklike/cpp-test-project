#pragma once
#include <string>
#include "test_engine.hpp"

class MainModule {
public:
    // API: начать тест
    void startTest(const std::string& filename);

    // API: завершить тест и показать результат
    void finishTest(int score, int total);
};
