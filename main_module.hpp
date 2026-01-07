#pragma once
#include <string>
#include "test_engine.hpp"
#include "auth_client.hpp"

class MainModule {
    AuthClient auth;
public:
    void startTest(const std::string& filename, const std::string& userId);
    void finishTest(int score, int total);
};
