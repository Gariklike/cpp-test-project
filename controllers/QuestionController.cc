#include <drogon/HttpController.h>

using namespace drogon;

class QuestionController : public drogon::HttpController<QuestionController> {
public:
    METHOD_LIST_BEGIN
        METHOD_ADD(QuestionController::getQuestions, "/questions", Get);
    METHOD_LIST_END

        void getQuestions(const HttpRequestPtr& req,
            std::function<void(const HttpResponsePtr&)>&& callback) {
        auto resp = HttpResponse::newHttpResponse();
        resp->setStatusCode(HttpStatusCode::k200OK);
        resp->setContentTypeCode(CT_APPLICATION_JSON);
        resp->addHeader("Access-Control-Allow-Origin", "*"); // разрешаем CORS

        resp->setBody(R"([
            {
                "id": 1,
                "question": "Сколько будет 2 + 2?",
                "answers": ["3", "4", "5"],
                "correctAnswer": 1
            },
            {
                "id": 2,
                "question": "Столица Франции?",
                "answers": ["Берлин", "Париж", "Рим"],
                "correctAnswer": 1
            },
            {
                "id": 3,
                "question": "Какой цвет получается при смешивании синего и жёлтого?",
                "answers": ["Зелёный", "Фиолетовый", "Оранжевый"],
                "correctAnswer": 0
            }
        ])");

        callback(resp);
    }
};
