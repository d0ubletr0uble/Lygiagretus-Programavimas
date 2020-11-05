package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

const DataCount = 30             // How much cars in json file
const RoutineCount = 10          // How much worker routines to start
const BufferSize = 5 // Size of DataMonitor internal buffer
const FilterCriteria = 26        // Select cars whose aging value is less

type (
	Car struct {
		Make    string  `json:"make"`
		Year    int     `json:"year"`
		Mileage float64 `json:"mileage"`
	}

	CarWithAge struct {
		Car Car
		Age int
	}
)

func main() {
	dataIn := make(chan Car)
	dataOut := make(chan Car)
	removeRequests := make(chan byte)
	addRequests := make(chan byte)

	workersOut := make(chan CarWithAge)
	workersToMain := make(chan byte)

	resultsOut := make(chan CarWithAge)

	cars := readData("IFK-8_OdinasT_L2_dat_3.json")
	startWorkers(dataOut, workersOut, removeRequests, workersToMain)
	go dataThread(dataIn, dataOut, addRequests, removeRequests)
	go resultThread(workersOut, resultsOut)
	fillDataThread(addRequests, dataIn, &cars)

	killWorkers(addRequests, dataIn, workersToMain)
	close(workersToMain)
	close(workersOut)

	// kill dataThread
	addRequests <- '-'
	close(dataIn)
	close(addRequests)
	close(removeRequests)

	results := getResults(resultsOut)
	writeData("IFK-8_OdinasT_L2_rez.txt", &cars, results)
}

func getResults(resultsOut <-chan CarWithAge) *[]CarWithAge {
	var results []CarWithAge
	for car := range resultsOut {
		results = append(results, car)
	}
	return &results
}

func killWorkers(addRequests chan<- byte, dataIn chan<- Car, workersToMain <-chan byte) {
	for i := 0; i < RoutineCount; i++ {
		addRequests <- '+'
		dataIn <- Car{Make: "<EndOfInput>"}
		<-workersToMain
	}
}

func fillDataThread(addRequests chan<- byte, dataIn chan<- Car, cars *[DataCount]Car) {
	for _, car := range cars {
		addRequests <- '+'
		dataIn <- car
	}
}

func startWorkers(dataOut chan Car, workersOut chan CarWithAge, removeRequests chan byte, workersToMain chan byte) {
	for i := 0; i < RoutineCount; i++ {
		go workerThread(dataOut, workersOut, removeRequests, workersToMain)
	}
}

func dataThread(in <-chan Car, out chan<- Car, requestIn <-chan byte, requestOut <-chan byte) {
	defer close(out)
	var cars [BufferSize]Car // circular buffer queue
	start := 0
	end := 0
	count := 0

	addCar := func() {
		cars[end] = <-in
		end = (end + 1) % BufferSize
		count++
	}

	removeCar := func() {
		out <- cars[start]
		start = (start + 1) % BufferSize
		count--
	}

	for {
		if count > 0 && count < BufferSize {
			// can add or remove
			select {
			case <-requestIn:
				addCar()
			case <-requestOut:
				removeCar()
			}
		} else if count == 0 {
			// can only add
			message := <-requestIn
			if message == '-' {
				break
			}
			addCar()
		} else {
			// can only remove
			<-requestOut
			removeCar()
		}
	}
}

func workerThread(in <-chan Car, out chan<- CarWithAge, requestData chan<- byte, done chan<- byte) {
	for {
		requestData <- '+'
		car := <-in
		if car.Make == "<EndOfInput>" {
			break // poison pill pattern
		}

		carAge := car.Age()
		if carAge < FilterCriteria {
			car := CarWithAge{car, carAge}
			out <- car
		}
	}
	done <- '+'
}

func resultThread(in <-chan CarWithAge, out chan<- CarWithAge) {
	defer close(out)
	var cars [DataCount]CarWithAge
	count := 0
	for car := range in {
		// insert sorted
		i := count - 1
		for i >= 0 && (cars[i].Age > car.Age || (cars[i].Age == car.Age && cars[i].Car.Year < car.Car.Year)) {
			cars[i+1] = cars[i]
			i--
		}
		cars[i+1] = car
		count++
	}

	// send data to Main
	for i := 0; i < count; i++ {
		out <- cars[i]
	}
}

// Computes custom car aging value
func (c *Car) Age() int {
	return time.Now().Year() - c.Year + int(c.Mileage/20_000)
}

func readData(path string) [DataCount]Car {
	data, _ := ioutil.ReadFile(path)
	var cars [DataCount]Car
	_ = json.Unmarshal(data, &cars)
	return cars
}

func writeData(path string, inputData *[DataCount]Car, results *[]CarWithAge) {
	file, _ := os.Create(path)
	i := 0
	defer file.Close()

	_, _ = fmt.Fprint(file, strings.Repeat("━", 42)+"\n")
	_, _ = fmt.Fprintf(file, "┃%25s%16s\n", "INPUT DATA", "┃")
	_, _ = fmt.Fprint(file, strings.Repeat("━", 42)+"\n")
	_, _ = fmt.Fprintf(file, "┃%-13s┃%10s┃%15s┃\n", "Make", "Year", "Mileage")
	_, _ = fmt.Fprint(file, strings.Repeat("━", 42)+"\n")
	for _, car := range inputData {
		_, _ = fmt.Fprintf(file, "%d┃%-13s┃%10d┃%15.2f┃\n",i, car.Make, car.Year, car.Mileage)
		i++
	}
	_, _ = fmt.Fprint(file, strings.Repeat("━", 42)+"\n\n")

	_, _ = fmt.Fprint(file, strings.Repeat("━", 48)+"\n")
	_, _ = fmt.Fprintf(file, "┃%29s%18s\n", "OUTPUT DATA", "┃")
	_, _ = fmt.Fprint(file, strings.Repeat("━", 48)+"\n")
	_, _ = fmt.Fprintf(file, "┃%-13s┃%10s┃%15s┃%5s┃\n", "Make", "Year", "Mileage", "Age")
	_, _ = fmt.Fprint(file, strings.Repeat("━", 48)+"\n")
	i=0
	for _, data := range *results {
		_, _ = fmt.Fprintf(file, "%d┃%-13s┃%10d┃%15.2f┃%5d┃\n",i, data.Car.Make, data.Car.Year, data.Car.Mileage, data.Age)
		i++
	}
	_, _ = fmt.Fprint(file, strings.Repeat("━", 48)+"\n")
}
