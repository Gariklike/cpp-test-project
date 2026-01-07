cmake_minimum_required(VERSION 3.5)
project(TestBackend)

set(CMAKE_CXX_STANDARD 17)

find_package(Drogon CONFIG REQUIRED)

add_executable(backend
    main.cpp
    controllers/ApiController.cc
)

target_link_libraries(backend PRIVATE Drogon::Drogon)
