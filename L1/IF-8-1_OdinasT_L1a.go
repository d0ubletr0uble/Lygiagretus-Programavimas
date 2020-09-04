package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/semaphore"
	"io/ioutil"
	"sync"
	"time"
)

// todo change to 30
const DataCount = 10    // How much cars in json file
const RoutineCount = 10 // How much worker routines to start
const BufferSize = DataCount / 2

type (
	Car struct {
		Make    string  `json:"make"`
		Year    int     `json:"year"`
		Mileage float64 `json:"mileage"`
	}

	CarWithHealth struct {
		Car    Car
		Health int
	}

	DataMonitor struct {
		Cars                  [BufferSize]Car
		In, Out               int
		Count, Space          semaphore.Weighted
		InputLock, OutputLock sync.Mutex
		context               context.Context  // needed for semaphore
	}

	SortedResultMonitor struct {
		Cars [DataCount]CarWithHealth
	}
)

func NewDataMonitor() *DataMonitor {
	ctx := context.TODO()
	monitor := DataMonitor{context: ctx}
	monitor.Space.Acquire(ctx, BufferSize)
	return &monitor
}

func (m *DataMonitor) addItem(car Car) { //todo maybe pass pointer?
	m.Space.Acquire(m.context, 1)
	m.InputLock.Lock()
	m.Cars[m.In] = car
	m.In = (m.In + 1) % BufferSize
	m.InputLock.Unlock()
	m.Count.Release(1)
}

func (m *DataMonitor) removeItem() Car {
	m.Count.Acquire(m.context, 1)
	m.OutputLock.Lock()
	car := m.Cars[m.Out]
	m.Out = (m.Out + 1) % BufferSize
	m.OutputLock.Unlock()
	m.Space.Release(1)
	return car
}

func main() {
	dataMonitor := NewDataMonitor()
	var resultMonitor SortedResultMonitor
	var waitGroup sync.WaitGroup
	waitGroup.Add(RoutineCount)

	// 1
	cars := readData("IF-8-1_OdinasT_L1_dat_1.json")

	// 2
	for i := 0; i < RoutineCount; i++ {
		go func() {
			worker(dataMonitor, &resultMonitor)
			waitGroup.Done()
		}()
	}

	// 3
	for _, car := range cars {
		dataMonitor.addItem(car)
	}

	// 4
	waitGroup.Wait()

	// 5
	writeData("IF-8-1_OdinasT_L1_rez.txt", resultMonitor)

}

func worker(in *DataMonitor, out *SortedResultMonitor) {
//todo take data from monitor 
	var car Car

	carAge := time.Now().Year() - car.Year + int(car.Mileage/20_000)
	fmt.Println(carAge)

}

func (m SortedResultMonitor) addItemSorted() {

}

func (m SortedResultMonitor) getItems() {

}

func readData(path string) [DataCount]Car {
	data, err := ioutil.ReadFile(path)

	if err != nil {
		panic(fmt.Sprintf("error when reading file %s", path))
	}

	var cars [DataCount]Car
	err = json.Unmarshal(data, &cars)

	if err != nil {
		panic(fmt.Sprintf("error converting json in file %s", path))
	}

	return cars
}

func writeData(path string, monitor SortedResultMonitor) {
	for _, car := range monitor.Cars {
		fmt.Println(car)
	}
}
