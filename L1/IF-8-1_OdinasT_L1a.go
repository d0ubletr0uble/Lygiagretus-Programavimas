package main

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/sync/semaphore"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

const DataCount = 30             // How much cars in json file
const RoutineCount = 10          // How much worker routines to start
const BufferSize = DataCount / 2 // Size of DataMonitor internal buffer
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

	DataMonitor struct {
		Cars                  [BufferSize]Car
		In, Out               int
		Count, Space          *semaphore.Weighted
		InputLock, OutputLock sync.Mutex
		Context               context.Context // needed for semaphore
	}

	SortedResultMonitor struct {
		Cars  [DataCount]CarWithAge
		Count int
		Lock  sync.Mutex
	}
)

func main() {
	dataMonitor := NewDataMonitor()
	resultMonitor := NewSortedResultMonitor()
	var wg sync.WaitGroup
	wg.Add(RoutineCount)

	cars := readData("IF-8-1_OdinasT_L1_dat_1.json")
	startWorkers(dataMonitor, resultMonitor, &wg)
	fillDataMonitor(&cars, dataMonitor)
	wg.Wait()
	writeData("IF-8-1_OdinasT_L1_rez.txt", &cars, resultMonitor)
}

func NewSortedResultMonitor() *SortedResultMonitor { return &SortedResultMonitor{} }

func NewDataMonitor() *DataMonitor {
	ctx := context.TODO()
	monitor := DataMonitor{
		Context: ctx,
		Count:   semaphore.NewWeighted(BufferSize),
		Space:   semaphore.NewWeighted(BufferSize),
	}

	// Exhaust semaphore so consumers couldn't start without data
	_ = monitor.Count.Acquire(ctx, BufferSize)
	return &monitor
}

func (m *DataMonitor) addItem(car Car) {
	_ = m.Space.Acquire(m.Context, 1)
	m.InputLock.Lock()
	m.Cars[m.In] = car
	m.In = (m.In + 1) % BufferSize
	m.InputLock.Unlock()
	m.Count.Release(1)
}

func (m *DataMonitor) removeItem() Car {
	_ = m.Count.Acquire(m.Context, 1)
	m.OutputLock.Lock()
	car := m.Cars[m.Out]

	if car.Make == "<EndOfInput>" { // Special message to close consumers when work is done
		m.Count.Release(1)
		m.OutputLock.Unlock()
		return car
	}

	m.Out = (m.Out + 1) % BufferSize
	m.OutputLock.Unlock()
	m.Space.Release(1)
	return car
}

// Computes custom car aging value
func (c *Car) Age() int {
	return time.Now().Year() - c.Year + int(c.Mileage/20_000)
}

func fillDataMonitor(cars *[DataCount]Car, dataMonitor *DataMonitor) {
	for _, car := range cars {
		dataMonitor.addItem(car) // Will block if no space
	}
	// Inform consumers that there will be no more data coming
	dataMonitor.addItem(Car{Make: "<EndOfInput>"})
}

func startWorkers(dataMonitor *DataMonitor, resultMonitor *SortedResultMonitor, wg *sync.WaitGroup) {
	for i := 0; i < RoutineCount; i++ {
		go func() {
			worker(dataMonitor, resultMonitor)
			wg.Done()
		}()
	}
}

func worker(in *DataMonitor, out *SortedResultMonitor) {
	for {
		car := in.removeItem() // Will block if no data
		if car.Make == "<EndOfInput>" {
			break
		}

		carAge := car.Age()
		if carAge < FilterCriteria {
			car := CarWithAge{car, carAge}
			out.addItemSorted(car)
		}
	}
}

func (m *SortedResultMonitor) addItemSorted(car CarWithAge) {
	m.Lock.Lock()
	i := m.Count - 1
	for i >= 0 && m.Cars[i].Age > car.Age {
		m.Cars[i+1] = m.Cars[i]
		i--
	}
	m.Cars[i+1] = car
	m.Count++
	m.Lock.Unlock()
}

func readData(path string) [DataCount]Car {
	data, _ := ioutil.ReadFile(path)
	var cars [DataCount]Car
	_ = json.Unmarshal(data, &cars)
	return cars
}

func writeData(path string, inputData *[DataCount]Car, results *SortedResultMonitor) {
	file, _ := os.Create(path)
	defer file.Close()

	_, _ = fmt.Fprint(file, strings.Repeat("━", 42)+"\n")
	_, _ = fmt.Fprintf(file, "┃%25s%16s\n", "INPUT DATA", "┃")
	_, _ = fmt.Fprint(file, strings.Repeat("━", 42)+"\n")
	_, _ = fmt.Fprintf(file, "┃%13s┃%10s┃%15s┃\n", "Make", "Year", "Mileage")
	_, _ = fmt.Fprint(file, strings.Repeat("━", 42)+"\n")
	for _, car := range inputData {
		_, _ = fmt.Fprintf(file, "┃%13s┃%10d┃%15.2f┃\n", car.Make, car.Year, car.Mileage)
	}
	_, _ = fmt.Fprint(file, strings.Repeat("━", 42)+"\n\n")

	_, _ = fmt.Fprint(file, strings.Repeat("━", 48)+"\n")
	_, _ = fmt.Fprintf(file, "┃%29s%18s\n", "OUTPUT DATA", "┃")
	_, _ = fmt.Fprint(file, strings.Repeat("━", 48)+"\n")
	_, _ = fmt.Fprintf(file, "┃%13s┃%10s┃%15s┃%5s┃\n", "Make", "Year", "Mileage", "Age")
	_, _ = fmt.Fprint(file, strings.Repeat("━", 48)+"\n")
	for i := 0; i < results.Count; i++ {
		data := results.Cars[i]
		_, _ = fmt.Fprintf(file, "┃%13s┃%10d┃%15.2f┃%5d┃\n", data.Car.Make, data.Car.Year, data.Car.Mileage, data.Age)
	}
	_, _ = fmt.Fprint(file, strings.Repeat("━", 48)+"\n")
}
