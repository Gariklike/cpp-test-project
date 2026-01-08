#pragma once
#include <string>

class AuthClient {
public:
    std::string getAccessToken(const std::string& code);
    bool hasPermission(const std::string& accessToken, const std::string& action);
};
