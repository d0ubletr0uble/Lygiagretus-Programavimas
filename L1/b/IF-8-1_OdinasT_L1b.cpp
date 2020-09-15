#include <iostream>
#include <omp.h>

using namespace std;

int main() {
#pragma omp parallel num_threads(10)
#pragma omp critical
        cout << "Hello, World!" << endl;

    return 0;
}
