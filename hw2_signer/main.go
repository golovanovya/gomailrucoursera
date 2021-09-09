package main

import (
	"sync"
)

func main() {
	println("test")
	jobs := []job{
		job(func(in, out chan interface{}) {
			out <- uint32(0)
			out <- uint32(1)
			out <- uint32(1)
			out <- uint32(2)
			out <- uint32(3)
			out <- uint32(5)
			out <- uint32(8)
		}),
		SingleHash,
		MultiHash,
		CombineResults,
		job(func(in, out chan interface{}) {
			for data := range in {
				println(data.(string))
				// out <- data
			}
		}),
	}
	in := make(chan interface{}, 3)
	wg := &sync.WaitGroup{}
	for _, job_item := range jobs {
		out := make(chan interface{}, 3)
		wg.Add(1)
		go func(in, out chan interface{}, job_item job) {
			defer wg.Done()
			defer close(out)
			job_item(in, out)
		}(in, out, job_item)
		in = out
	}
	wg.Wait()
}
