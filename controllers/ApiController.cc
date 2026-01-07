#include "ApiController.h"
#include <fstream>
#include <json/json.h>
#include <drogon/drogon.h>

static const std::string QUESTIONS_FILE = "data/questions.json";
static const std::string RESULTS_FILE = "results.json";

void ApiController::getQuestions(const HttpRequestPtr& req,
    std::function<void(const HttpResponsePtr&)>&& callback) {
    Json::Value data;
    std::ifstream file(QUESTIONS_FILE);
    if (!file.is_open()) {
        LOG_ERROR << "Не удалось открыть файл: " << QUESTIONS_FILE;
        auto resp = HttpResponse::newHttpJsonResponse(Json::Value("Файл вопросов не найден"));
        resp->setStatusCode(k500InternalServerError);
        callback(resp);
        return;
    }

    try {
        file >> data;
    }
    catch (const std::exception& e) {
        LOG_ERROR << "Ошибка чтения JSON: " << e.what();
        auto resp = HttpResponse::newHttpJsonResponse(Json::Value("Ошибка чтения JSON"));
        resp->setStatusCode(k500InternalServerError);
        callback(resp);
        return;
    }

    if (data.isNull() || !data.isArray()) {
        LOG_ERROR << "Некорректный формат JSON в " << QUESTIONS_FILE;
        auto resp = HttpResponse::newHttpJsonResponse(Json::Value("Некорректный формат данных"));
        resp->setStatusCode(k500InternalServerError);
        callback(resp);
        return;
    }

    auto resp = HttpResponse::newHttpJsonResponse(data);
    callback(resp);
}

void ApiController::postResults(const HttpRequestPtr& req,
    std::function<void(const HttpResponsePtr&)>&& callback) {
    auto json = req->getJsonObject();
    if (!json) {
        auto resp = HttpResponse::newHttpJsonResponse(Json::Value("Неверный JSON"));
        resp->setStatusCode(k400BadRequest);
        callback(resp);
        return;
    }

    Json::Value results;
    std::ifstream fileIn(RESULTS_FILE);
    if (fileIn.good()) {
        try {
            fileIn >> results;
        }
        catch (...) {
            results = Json::Value(Json::arrayValue);
        }
    }
    fileIn.close();

    results.append(*json);

    std::ofstream fileOut(RESULTS_FILE);
    fileOut << results;

    auto resp = HttpResponse::newHttpJsonResponse(Json::Value("Сохранено"));
    callback(resp);
}

void ApiController::getResult(const HttpRequestPtr& req,
    std::function<void(const HttpResponsePtr&)>&& callback) {
    auto userId = req->getParameter("userId");

    Json::Value results;
    std::ifstream file(RESULTS_FILE);
    if (!file.is_open()) {
        auto resp = HttpResponse::newHttpJsonResponse(Json::Value("Файл результатов не найден"));
        resp->setStatusCode(k500InternalServerError);
        callback(resp);
        return;
    }

    try {
        file >> results;
    }
    catch (...) {
        results = Json::Value(Json::arrayValue);
    }

    Json::Value userResults(Json::arrayValue);
    for (const auto& r : results) {
        if (r.isMember("userId") && r["userId"].asString() == userId) {
            userResults.append(r);
        }
    }

    auto resp = HttpResponse::newHttpJsonResponse(userResults);
    callback(resp);
}
