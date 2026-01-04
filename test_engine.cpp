#include "test_engine.hpp"
#include "json.hpp"
#include <fstream>
#include <iostream>
#include <stdexcept>
#include <limits>

using json = nlohmann::json;

std::vector<Question> TestEngine::loadQuestions(const std::string& filename) {
    std::ifstream file(filename);
    if (!file.is_open()) throw std::runtime_error("Не удалось открыть файл: " + filename);

    json j;
    file >> j;

    std::vector<Question> questions;
    for (auto& item : j) {
        Question q;
        q.text = item["text"];
        q.options = item["options"].get<std::vector<std::string>>();
        q.correct = item["correct"];
        questions.push_back(q);
    }
    return questions;
}

static int safeInput(int min, int max) {
    int value;
    while (true) {
        std::cin >> value;
        if (std::cin.fail() || value < min || value > max) {
            std::cout << "Введите число от " << min << " до " << max << ": ";
            std::cin.clear();
            std::cin.ignore(std::numeric_limits<std::streamsize>::max(), '\n');
        } else return value;
    }
}

int TestEngine::run(const std::vector<Question>& questions) {
    int score = 0;
    for (size_t i = 0; i < questions.size(); i++) {
        const auto& q = questions[i];
        std::cout << "\nВопрос " << i+1 << ": " << q.text << "\n";
        for (size_t k = 0; k < q.options.size(); k++) {
            std::cout << k+1 << ") " << q.options[k] << "\n";
        }
        std::cout << "Ваш ответ: ";
        int answer = safeInput(1, q.options.size());
        if (answer-1 == q.correct) score++;
    }
    return score;
}

void TestEngine::saveResult(int score, int total, const std::string& filename) {
    json result = {
        {"score", score},
        {"total", total},
        {"percent", (int)((score * 100.0) / total)}
    };
    std::ofstream out(filename);
    out << result.dump(2);
}
