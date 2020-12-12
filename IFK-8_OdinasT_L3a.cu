
#include "cuda_runtime.h"
#include "device_launch_parameters.h"

#include <stdio.h>
#include <iostream>
#include <fstream>
#include <string>
#include <sstream>
#include <ctime>
using namespace std;

// CONFIGURATION
const char* DATA_FILE = "IFK-8_OdinasT_L3_dat2.csv";
const int CAR_COUNT = 30;
const int RESULT_SIZE = 13;
const int THREAD_COUNT = 7;

const int GRADE_A_THRESHOLD = 15;
const int GRADE_B_THRESHOLD = 25;
//--

struct Car
{
	char Make[11];
	int Year;
	float Mileage;
};

void readData(const char* path, Car *cars)
{
	int i = 0;
	ifstream stream(path);
	string line;
	
	while (getline(stream, line))
	{
		istringstream str(line);
		string token;
		getline(str, token, ',');
		string make = token;
		getline(str, token, ',');
		int year = stoi(token);
		getline(str, token, ',');
		float mileage = stof(token);
		struct Car c;
		strcpy(c.Make, make.c_str());
		c.Year = year;
		c.Mileage = mileage;
		cars[i++] = c;
	}
	
	stream.close();
}

int getYear()
{
	time_t now = time(0);
	tm *ltm = localtime(&now);
	return 1900 + ltm->tm_year;
}

__global__ void run_on_gpu(Car *cars, int *year, char *results, int *result_count)
{
    int thread_count = blockDim.x;
	int i = threadIdx.x;

	while(i < CAR_COUNT)
	{
		int age = *year - cars[i].Year + cars[i].Mileage / 20000;
		char grade;
		
		if(age <= GRADE_A_THRESHOLD)
			grade = 'A';
		else if(age <= GRADE_B_THRESHOLD)
			grade = 'B';
		else
			grade = 'C';
			
		if (grade != 'C')
		{
			int index = atomicAdd(result_count, 1);
			index *= RESULT_SIZE;
			for (int j = 0; cars[i].Make[j] != 0; j++, index++)
				results[index] = cars[i].Make[j];
			results[index] = '-';
			index++;
			results[index] = grade;
		}
		i += thread_count;
	}
}

void writeResults(const char *path, const char *results, int count)
{
	ofstream stream(path);
	for (int i = 0; i < count * RESULT_SIZE; i++)
		stream << results[i];
	stream.close();
}

int main()
{
	int year = getYear(); //current year
	struct Car cars[CAR_COUNT];
	int result_count = 0;
	
	Car *device_cars;
	int *device_year;
	char *device_results;
	int *device_result_count;
	
	readData(DATA_FILE, cars);

	cudaMalloc(&device_cars, sizeof(cars));
	cudaMalloc(&device_year, sizeof(int));
	cudaMalloc(&device_results, sizeof(char) * RESULT_SIZE * CAR_COUNT);
	cudaMalloc(&device_result_count, sizeof(int));
	
	//copy data to GPU memory
	cudaMemcpy(device_cars, cars, sizeof(cars), cudaMemcpyHostToDevice);
	cudaMemcpy(device_year, &year, sizeof(int), cudaMemcpyHostToDevice);
	cudaMemcpy(device_result_count, &result_count, sizeof(int), cudaMemcpyHostToDevice);
	
	run_on_gpu<<< 1, THREAD_COUNT >>>(device_cars, device_year, device_results, device_result_count);
	cudaDeviceSynchronize();

	//get results back from GPU
	char results[RESULT_SIZE*CAR_COUNT];
	cudaMemcpy(results, device_results, sizeof(results), cudaMemcpyDeviceToHost);
	cudaMemcpy(&result_count, device_result_count, sizeof(int), cudaMemcpyDeviceToHost);
	
	writeResults("IFK-8_OdinasT_L3_rez.txt", results, result_count);

	//release GPU memory
	cudaFree(device_cars);
	cudaFree(device_year);
	cudaFree(device_results);
	cudaFree(device_result_count);
	
}