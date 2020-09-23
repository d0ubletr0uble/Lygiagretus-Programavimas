#include <iostream>
#include <iomanip>
#include <fstream>
#include <omp.h>
#include <nlohmann/json.hpp>
#include <utility>
#include <ctime>

using namespace std;
using json = nlohmann::json;

const int DATA_COUNT = 30;
const int BUFFER_SIZE = DATA_COUNT / 2;
const int FILTER_CRITERIA = 26;
const int THREAD_COUNT = 10;

class Car {
public:
    string Make;
    int Year;
    double Mileage;

    int GetAge() const {
        time_t now = time(0);
        tm *ltm = localtime(&now);
        int year = 1900 + ltm->tm_year;
        return year - Year + (int) (Mileage / 20000);
    }

    Car(string make, int year, double mileage) : Make(std::move(make)), Year(year), Mileage(mileage) {}

    explicit Car(const json &json) : Make(json["make"].get<string>()), Year(json["year"].get<int>()),
                                     Mileage(json["mileage"].get<double>()) {}

    Car() = default;

};

class CarWithAge {
public:
    Car car;
    int Age;

    CarWithAge(Car c, int age) : car(c), Age(age) {}

    CarWithAge() = default;
};

class DataMonitor {
    Car cars[BUFFER_SIZE];
    int count = 0;
    bool finished = false;

public:
    void Finished() { finished = true; }

    int TryPush(const Car &car) {
        int status = 0;
        #pragma omp critical
        {
            if (count < BUFFER_SIZE) { // buffer has space
                cars[count++] = car;
                status = 1;
            }
        }
        return status;
    }

    tuple<int, Car> TryPop() {
        Car car;
        int status = 0;
        #pragma omp critical
        {
            if (count > 0) { // buffer has data
                car = cars[--count];
                status = 1;
            } else if (finished)
                status = -1;
        }
        return {status, car};
    }

    void Fill(Car (&cars)[DATA_COUNT]) {
        for (const Car &car: cars)
            while (!TryPush(car)) // forcefully push until success
                continue;
        Finished();
    }
};

class ResultMonitor {
public:
    CarWithAge Cars[DATA_COUNT];
    int Count = 0;

    void InsertSorted(const CarWithAge &car) {
        #pragma omp critical (results)
        {
            int i = Count - 1;
            while (i >= 0 && (Cars[i].Age > car.Age || (Cars[i].Age == car.Age && Cars[i].car.Year < car.car.Year))) {
                Cars[i + 1] = Cars[i];
                i--;
            }
            Cars[i + 1] = car;
            Count++;
        }
    }

    void WriteDataToFile(const string &file, Car (&inputData)[DATA_COUNT]) {
        ofstream stream(file);

        stream << string (42, '-') << endl;
        stream << "|" << setw(25) << "INPUT DATA" << setw(16) << '|' << endl;
        stream << string (42, '-') << endl;
        stream << setw(14) << left << "|Make" << "|" << right << setw(10) << "Year|" << setw(17) << "Mileage|" << endl;
        stream << string (42, '-') << endl;

        for(const Car &car: inputData)
            stream << "|" << setw(13) << left << car.Make << right << "|"  << setw(9) << car.Year << "|" << setw(16) << fixed << setprecision(2) << car.Mileage << "|" << endl;
        stream << string (42, '-') << endl << endl;

        stream << string (48, '-') << endl;
        stream << "|" << setw(29) << "OUTPUT DATA" << setw(18) << '|' << endl;
        stream << string (48, '-') << endl;
        stream << setw(14) << left << "|Make" << "|" << right << setw(10) << "Year|" << setw(17) << "Mileage|"  << setw(6) << "Age|"<< endl;
        stream << string (48, '-') << endl;
        for (int i = 0; i < Count; ++i) {
            const CarWithAge &c = Cars[i];
            stream << "|" << setw(13) << left << c.car.Make << right << "|"  << setw(9) << c.car.Year << "|" << setw(17) << fixed << setprecision(2) << c.car.Mileage << "|" << setw(4) << c.Age << "|" << endl;
        }
        stream << string (48, '-') << endl;

        stream.close();
    }

};

void ReadData(const string &file, Car (&cars)[DATA_COUNT]) {
    int i = 0;
    ifstream stream(file);
    auto json = json::parse(stream);
    for (const auto &j: json) {
        cars[i] = Car(j);
        i++;
    }
    stream.close();
}

void StartWorker(DataMonitor &dataMonitor, ResultMonitor &resultMonitor) {
    while (true) {
        auto[status, car] = dataMonitor.TryPop();

        if (status == 1) {
            int age = car.GetAge();
            if (age < FILTER_CRITERIA)
                resultMonitor.InsertSorted(CarWithAge(car, age));

        } else if (status == -1)
            break;
    }
}

int main() {
    DataMonitor dataMonitor;
    ResultMonitor resultMonitor;
    Car cars[DATA_COUNT];
    ReadData("../../IFK-8_OdinasT_L1_dat_1.json", cars);

    #pragma omp parallel num_threads(THREAD_COUNT) shared(cars, dataMonitor, resultMonitor) default(none)
    {
    #pragma omp task shared(dataMonitor, resultMonitor) default(none)
        StartWorker(dataMonitor, resultMonitor);

    #pragma omp master
        dataMonitor.Fill(cars);
    }

    resultMonitor.WriteDataToFile("../../IFK-8_OdinasT_L1_rez.txt", cars);
    return 0;
}
