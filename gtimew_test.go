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
	"math/rand"
	"testing"
	"time"
)

func TestNewGTimeW(t *testing.T) {
	inf := make(chan bool)
	gtw := NewGTimeW()
	gtw.Start()
	defer gtw.Stop()

	type param struct {
		name   string
		preSet int
	}
	st := time.Now()
	for i := 0; i < 100; i++ {
		// random schedule two minutes
		delaySec := rand.Intn(121)
		name := "task" + fmt.Sprint(i)
		gtw.AddTask(delaySec, name, func(i interface{}) {
			p := i.(*param)
			fmt.Printf("%s has been scheduled after %v, pre-set is %v sec.\n", p.name, time.Since(st), p.preSet)
		}, &param{name: name, preSet: delaySec})
	}

	<-inf
}

func BenchmarkGTimeW(b *testing.B) {
	gtw := NewGTimeW()
	gtw.Start()
	defer gtw.Stop()
	for i := 0; i < b.N; i++ {
		gtw.AddTask(3, "task"+fmt.Sprint(i), func(i interface{}) {

		}, nil)
	}
}

func BenchmarkTimer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = time.AfterFunc(3*time.Second, func() {
		})
	}
}
