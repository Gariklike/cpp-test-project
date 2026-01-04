#pragma once
#include <string>
#include <vector>

// Структура вопроса
struct Question {
    std::string text;
    std::vector<std::string> options;
    int correct;
};

// Класс движка теста
class TestEngine {
public:
    // Загрузка вопросов из JSON
    static std::vector<Question> loadQuestions(const std::string& filename);

    // Запуск теста (возвращает количество правильных ответов)
    int run(const std::vector<Question>& questions);

    // Сохранение результата
    void saveResult(int score, int total, const std::string& filename = "result.json");
};
