package main

import (
	"fmt"
	"testing"
	"time"
)

func TestNewGTimeW(t *testing.T) {
	inf := make(chan bool)
	gtw, _ := NewGTimeW(time.Second, 60, nil)
	gtw.Start()
	defer gtw.Stop()

	st := time.Now()
	//gtw.AddTask(3, "task1", func(i interface{}) {
	//	fmt.Printf("task1 scheduled after %v\n", time.Since(st))
	//}, nil)

	gtw.AddTask(time.Second, "task2", func(i interface{}) {
		fmt.Printf("task2 scheduled after %v\n", time.Since(st))
	}, nil)

	<-inf
}

func BenchmarkGTimeW(b *testing.B) {
	gtw, _ := NewGTimeW(time.Second, 60, nil)
	gtw.Start()
	defer gtw.Stop()
	for i := 0; i < b.N; i++ {
		gtw.AddTask(3*time.Second, "task"+fmt.Sprint(i), func(i interface{}) {

		}, nil)
	}
}

func BenchmarkTimer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = time.AfterFunc(3*time.Second, func() {
		})
	}
}
