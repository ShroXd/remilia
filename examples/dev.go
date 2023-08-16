package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	done := make(chan interface{})
	defer close(done)

	start := time.Now()
	randIntStream := toInt(done, repeatFn(done, random))
	fmt.Println("Primes: ")
	for prime := range take(done, primeFinder(done, randIntStream), 10) {
		fmt.Printf("\t%d\n", prime)
	}

	fmt.Printf("Search took: %v", time.Since(start))
}

func random() interface{} {
	return rand.Intn(5000000000000000000)
}

func repeatFn(
	done <-chan interface{},
	fn func() interface{},
) <-chan interface{} {
	valueStream := make(chan interface{})

	go func() {
		defer close(valueStream)

		for {
			select {
			case <-done:
				return
			case valueStream <- fn():
			}
		}
	}()

	return valueStream
}

func toInt(
	done <-chan interface{},
	valueStream <-chan interface{},
) <-chan int {
	intStream := make(chan int)

	go func() {
		defer close(intStream)

		for v := range valueStream {
			select {
			case <-done:
				return
			case intStream <- v.(int):
			}
		}
	}()

	return intStream
}

func take(
	done <-chan interface{},
	valueStream <-chan int,
	num int,
) <-chan interface{} {
	takeStream := make(chan interface{})

	go func() {
		defer close(takeStream)

		for i := 0; i < num; i++ {
			select {
			case <-done:
				return
			case takeStream <- <-valueStream:
			}
		}
	}()

	return takeStream
}

func isPrime(num int) bool {
	if num <= 1 {
		return false
	}

	for i := 2; i*i <= num; i++ {
		if num%i == 0 {
			return false
		}
	}

	return true
}

func primeFinder(
	done <-chan interface{},
	intStream <-chan int,
) <-chan int {
	primeStream := make(chan int)

	go func() {
		defer close(primeStream)

		for v := range intStream {
			if isPrime(v) {
				select {
				case <-done:
					return
				case primeStream <- v:
				}
			}
		}
	}()

	return primeStream
}
