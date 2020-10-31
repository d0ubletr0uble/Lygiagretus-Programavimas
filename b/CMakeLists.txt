cmake_minimum_required(VERSION 3.17)
project(b)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -std=c++11 -fopenmp")


add_executable(b IFK-8_OdinasT_L1b.cpp)