package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

var mu sync.Mutex

func syncDataSignerMd5(data string) string {
	mu.Lock()
	defer mu.Unlock()
	return DataSignerMd5(data)
}

func asyncDataSignerMd5(data string) chan string {
	out := make(chan string, 1)
	go func() {
		defer close(out)
		result := syncDataSignerMd5(data)
		println(data, "md5", result)
		out <- result
	}()
	return out
}

func asyncDataSignerCrc32(data string) chan string {
	out := make(chan string, 1)
	go func() {
		defer close(out)
		out <- DataSignerCrc32(data)
	}()
	return out
}

func asyncDataSignerMd5Crc32(data string) chan string {
	out := make(chan string, 1)
	go func() {
		defer close(out)
		out <- <-asyncDataSignerCrc32(<-asyncDataSignerMd5(data))
	}()
	return out
}

func asyncConcatCrc32Md5Crc32(data string) chan string {
	out := make(chan string, 1)
	go func() {
		defer close(out)
		first, second := asyncDataSignerCrc32(data), asyncDataSignerMd5Crc32(data)
		out <- <-first + "~" + <-second
	}()
	return out
}

func format(data interface{}) string {
	switch data.(type) {
	case int:
		return fmt.Sprintf("%d", data.(int))
	case uint32:
		return fmt.Sprintf("%d", data.(uint32))
	default:
		panic("undefined type")
	}
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for data := range in {
		wg.Add(1)
		go func(out chan interface{}, val string, wg *sync.WaitGroup) {
			defer wg.Done()
			out <- <-asyncConcatCrc32Md5Crc32(val)
		}(out, format(data), wg)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for data := range in {
		wg.Add(1)
		go func(out chan interface{}, val string, wg *sync.WaitGroup) {
			defer wg.Done()
			result := make([]string, 6)
			resultChan := make(chan chan string, 6)
			for i := 0; i < 6; i++ {
				resultChan <- asyncDataSignerCrc32(fmt.Sprintf("%d%s", i, val))
			}
			for i := 0; i < 6; i++ {
				result[i] = <-<-resultChan
			}
			out <- strings.Join(result[:], "")
		}(out, data.(string), wg)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var result []string
	for data := range in {
		val := data.(string)
		println(val)
		result = append(result, val)
	}
	sort.Strings(result)
	out <- strings.Join(result[:], "_")
}

func ExecutePipeline(jobs ...job) {
	in := make(chan interface{}, 1)
	wg := &sync.WaitGroup{}
	for _, job_item := range jobs {
		out := make(chan interface{}, 3)
		wg.Add(1)
		go func(in, out chan interface{}, job_item job, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)
			job_item(in, out)
		}(in, out, job_item, wg)
		in = out
	}
	wg.Wait()
}
