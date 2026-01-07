#pragma once
#include <drogon/HttpController.h>

using namespace drogon;

class ApiController : public drogon::HttpController<ApiController> {
public:
    METHOD_LIST_BEGIN
        ADD_METHOD_TO(ApiController::getQuestions, "/questions", Get);
        ADD_METHOD_TO(ApiController::postResults, "/results", Post);
        ADD_METHOD_TO(ApiController::getResult, "/result", Get);
    METHOD_LIST_END

    void getQuestions(const HttpRequestPtr &req,
                      std::function<void(const HttpResponsePtr &)> &&callback);

    void postResults(const HttpRequestPtr &req,
                     std::function<void(const HttpResponsePtr &)> &&callback);

    void getResult(const HttpRequestPtr &req,
                   std::function<void(const HttpResponsePtr &)> &&callback);
};
