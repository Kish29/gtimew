/*
   Copyright [yyyy] [name of copyright owner]

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

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
