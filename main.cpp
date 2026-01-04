#include <iostream>
#include <fstream>
#include <vector>
#include <string>
#include <stdexcept>
#include <chrono>
#include <limits>
#include "json.hpp"

using json = nlohmann::json;

// ---------------------------
// Структура вопроса
// ---------------------------
struct Question {
    std::string text;
    std::vector<std::string> options;
    int correct;
};

// ---------------------------
// Загрузка вопросов из JSON
// ---------------------------
std::vector<Question> load_questions(const std::string& filename) {
    std::ifstream file(filename);

    if (!file.is_open()) {
        throw std::runtime_error("Не удалось открыть файл: " + filename);
    }

    json j;
    try {
        file >> j;
    } catch (...) {
        throw std::runtime_error("Ошибка: некорректный JSON в файле " + filename);
    }

    std::vector<Question> questions;

    for (auto& item : j) {
        if (!item.contains("text") || !item.contains("options") || !item.contains("correct")) {
            throw std::runtime_error("Ошибка: один из вопросов имеет неверный формат");
        }

        Question q;
        q.text = item["text"];
        q.options = item["options"].get<std::vector<std::string>>();
        q.correct = item["correct"];

        questions.push_back(q);
    }

    return questions;
}

// ---------------------------
// Безопасный ввод числа
// ---------------------------
int safe_input(int min, int max) {
    int value;

    while (true) {
        std::cin >> value;

        if (std::cin.fail() || value < min || value > max) {
            std::cout << "Введите число от " << min << " до " << max << ": ";
            std::cin.clear();
            std::cin.ignore(std::numeric_limits<std::streamsize>::max(), '\n');
        } else {
            return value;
        }
    }
}

// ---------------------------
// Основная логика теста
// ---------------------------
int run_test(const std::vector<Question>& questions) {
    int score = 0;

    for (size_t i = 0; i < questions.size(); i++) {
        const auto& q = questions[i];

        std::cout << "\n-----------------------------\n";
        std::cout << "Вопрос " << i + 1 << " из " << questions.size() << "\n";
        std::cout << q.text << "\n";

        for (size_t k = 0; k < q.options.size(); k++) {
            std::cout << k + 1 << ") " << q.options[k] << "\n";
        }

        std::cout << "Ваш ответ: ";
        int answer = safe_input(1, q.options.size());

        if (answer - 1 == q.correct) {
            score++;
        }
    }

    return score;
}

// ---------------------------
// Сохранение результата
// ---------------------------
void save_result(int score, int total) {
    json result = {
        {"score", score},
        {"total", total},
        {"percent", (int)((score * 100.0) / total)}
    };

    std::ofstream out("result.json");
    out << result.dump(2);

    std::cout << "\nРезультат сохранён в result.json\n";
}

// ---------------------------
// main()
// ---------------------------
int main(int argc, char* argv[]) {
    setlocale(LC_ALL, "Russian");

    std::string filename = "questions.json";
    if (argc > 1) {
        filename = argv[1];
    }

    try {
        auto questions = load_questions(filename);

        std::cout << "Загружено вопросов: " << questions.size() << "\n";
        std::cout << "Начинаем тест...\n";

        int score = run_test(questions);

        std::cout << "\nВаш результат: " << score << " из " << questions.size() << "\n";
        std::cout << "Процент: " << (score * 100.0 / questions.size()) << "%\n";

        save_result(score, questions.size());
    }
    catch (const std::exception& e) {
        std::cout << "\nОшибка: " << e.what() << "\n";
        return 1;
    }

    return 0;
}
