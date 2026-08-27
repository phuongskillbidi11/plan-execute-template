cmake_minimum_required(VERSION 3.16)
project(EmbeddedQtApp LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Minimal fixture stub representing a Qt application cross-compiled for an
# ARM embedded Linux target — see Phase 9
# benchmarks/scenarios/qt-cmake-embedded-linux.yaml. No real Qt/toolchain
# dependency is required to exist for eng's own stack detection.
add_executable(embedded_qt_app main.cpp)
