#include <drogon/drogon.h>

int main() {
    drogon::app()
        .registerHandler("/questions", [](const HttpRequestPtr& req,
            std::function<void(const HttpResponsePtr&)>&& callback) {
                auto resp = HttpResponse::newHttpResponse();
                resp->setStatusCode(HttpStatusCode::k200OK);
                resp->setContentTypeCode(CT_APPLICATION_JSON);
                resp->addHeader("Access-Control-Allow-Origin", "*");

                resp->setBody(R"([
                {
                    "id": 1,
                    "question": "Сколько будет 2 + 2?",
                    "answers": ["3", "4", "5"],
                    "correctAnswer": 1
                }
            ])");

                callback(resp);
            });

    drogon::app().run();
}
