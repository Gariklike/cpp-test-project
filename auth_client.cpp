#include "auth_client.hpp"
#include <curl/curl.h>
#include <json.hpp>
#include <string>
#include <iostream>
#include <vector>
#include <algorithm>

using json = nlohmann::json;

static size_t WriteCallback(void* contents, size_t size, size_t nmemb, void* userp) {
    ((std::string*)userp)->append((char*)contents, size * nmemb);
    return size * nmemb;
}

std::string AuthClient::getAccessToken(const std::string& code) {
    CURL* curl = curl_easy_init();
    std::string response;

    if (curl) {
        std::string url = "http://localhost:8000/auth/code/verify";
        std::string payload = "{\"code\":\"" + code + "\"}";

        struct curl_slist* headers = nullptr;
        headers = curl_slist_append(headers, "Content-Type: application/json");

        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, payload.c_str());
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);

        CURLcode res = curl_easy_perform(curl);
        curl_easy_cleanup(curl);
        curl_slist_free_all(headers);

        if (res == CURLE_OK) {
            try {
                auto jsonResponse = json::parse(response);
                return jsonResponse.value("access_token", "");
            } catch (...) {
                std::cerr << "Ошибка парсинга JSON\n";
            }
        } else {
            std::cerr << "Ошибка запроса: " << curl_easy_strerror(res) << "\n";
        }
    }

    return "";
}

bool AuthClient::hasPermission(const std::string& accessToken, const std::string& action) {
    CURL* curl = curl_easy_init();
    std::string response;

    if (curl) {
        std::string url = "http://localhost:8000/token/validate";
        std::string payload = "{\"access_token\":\"" + accessToken + "\"}";

        struct curl_slist* headers = nullptr;
        headers = curl_slist_append(headers, "Content-Type: application/json");

        curl_easy_setopt(curl, CURLOPT_URL, url.c_str());
        curl_easy_setopt(curl, CURLOPT_POSTFIELDS, payload.c_str());
        curl_easy_setopt(curl, CURLOPT_HTTPHEADER, headers);
        curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, WriteCallback);
        curl_easy_setopt(curl, CURLOPT_WRITEDATA, &response);

        CURLcode res = curl_easy_perform(curl);
        curl_easy_cleanup(curl);
        curl_slist_free_all(headers);

        if (res == CURLE_OK) {
            try {
                auto jsonResponse = json::parse(response);
                if (jsonResponse.value("valid", false)) {
                    auto permissions = jsonResponse.value("permissions", std::vector<std::string>{});
                    return std::find(permissions.begin(), permissions.end(), action) != permissions.end();
                }
            } catch (...) {
                std::cerr << "Ошибка парсинга JSON\n";
            }
        } else {
            std::cerr << "Ошибка запроса: " << curl_easy_strerror(res) << "\n";
        }
    }

    return false;
}
