package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Buffer interface {
	Push(el int)
	Get() []int
}

type RingIntBuffer struct {
	array []int
	pos   int
	size  int
	mx    sync.RWMutex
}

// Интервал очистки кольцевого буфера
const bufferDrainInterval = 10 * time.Second

// Размер кольцевого буфера
const bufferSize int = 7

func main() {

	fmt.Println("Для заполнения буфера введите целые числа ...")

	input := make(chan int)
	done := make(chan bool)

	go Read(input, done)

	negativeFiltrCh := make(chan int)
	go NegativeFiltrStageInt(input, negativeFiltrCh, done)

	notDivadedThreeCh := make(chan int)
	go NotDivadedThreeFunc(negativeFiltrCh, notDivadedThreeCh, done)

	bufferedIntCh := make(chan int)

	go BufferStageFunc(notDivadedThreeCh, bufferedIntCh, done, bufferSize, bufferDrainInterval)

	for {
		select {

		case data := <-bufferedIntCh:
			fmt.Println("Получены данные: ... ", data)
		case <-done:
			return
		}
	}
}

// NewRingIntBuffer Конструктор
func NewRingIntBuffer(size int) Buffer {
	return &RingIntBuffer{make([]int, size), -1, size, sync.RWMutex{}}
}

// Push добавление нового элемента в конец буфера
// При попытке добавления нового элемента в заполненный буфер
// самое старое значение затирается
func (r *RingIntBuffer) Push(el int) {
	r.mx.Lock()
	defer r.mx.Unlock()

	if r.pos == r.size-1 {
		for i := 1; i <= r.size-1; i++ {
			r.array[i-1] = r.array[i]
		}
		r.array[r.pos] = el
	} else {
		r.pos++
		r.array[r.pos] = el
	}
}

// Get - получение всех элементов буфера и его последующая очистка
func (r *RingIntBuffer) Get() []int {
	if r.pos <= 0 {
		return nil
	}
	r.mx.Lock()
	defer r.mx.Unlock()
	var output = r.array[:r.pos+1]

	r.pos = -1
	return output
}

// Read Чтение с консоли
func Read(nextStage chan<- int, done chan bool) {
	scanner := bufio.NewScanner(os.Stdin)
	var data string

	for scanner.Scan() {
		data = scanner.Text()

		if strings.EqualFold(data, "Выход") { // Пишем в консоли 'Выход' для завершения программы.
			fmt.Print("\nПрограмма завершила работу!")
			close(done)
			return
		}
		i, err := strconv.Atoi(data)
		if err != nil {
			fmt.Print("Программа обрабатывает только целые числа.\n\n Или введите команду :\n  Выход \n")
			continue
		}
		nextStage <- i
	}
}

// NegativeFiltrStageInt Фильтрует отрицательные числа и равные 0
func NegativeFiltrStageInt(previousStageCh <-chan int, nextStageCh chan<- int, done <-chan bool) {
	for {
		select {

		case data := <-previousStageCh:
			if data > 0 {
				nextStageCh <- data
			}
		case <-done:
			return
		}
	}
}

// NotDivadedThreeFunc Фильтрует числа кратные 3
func NotDivadedThreeFunc(previusStageCh <-chan int, nextStageCh chan<- int, done <-chan bool) {
	for {
		select {

		case data := <-previusStageCh:
			if data%3 != 0 {
				nextStageCh <- data
			}
		case <-done:
			return
		}
	}
}

// BufferStageFunc Функция выполняющая опустошение буфера с заданным
// интервалом
func BufferStageFunc(previusStageCh <-chan int, nextStageCh chan<- int, done <-chan bool, size int, interval time.Duration) {
	buffer := NewRingIntBuffer(size)
	for {
		select {

		case data := <-previusStageCh:
			buffer.Push(data)
		case <-time.After(interval):
			bufferData := buffer.Get()

			if bufferData != nil {
				for _, data := range bufferData {
					nextStageCh <- data
				}
			}
		case <-done:
			return
		}
	}
}
