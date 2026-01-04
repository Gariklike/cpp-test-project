#include "main_module.hpp"
#include <iostream>

void MainModule::startTest(const std::string& filename) {
    auto questions = TestEngine::loadQuestions(filename);
    std::cout << "Загружено вопросов: " << questions.size() << "\n";
    TestEngine engine;
    int score = engine.run(questions);
    finishTest(score, questions.size());
    engine.saveResult(score, questions.size());
}

void MainModule::finishTest(int score, int total) {
    std::cout << "\nВаш результат: " << score << " из " << total << "\n";
    std::cout << "Процент: " << (score * 100.0 / total) << "%\n";
}
