package main

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maxfer4maxfer/goDebuger"
)

// ExecutePipeline execute input jobs after each other
func ExecutePipeline(jobs ...job) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	in := make(chan interface{})

	for _, jobItem := range jobs {
		out := make(chan interface{})
		wg.Add(1)
		go func(jobFunc job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()

			defer close(out)
			jobFunc(in, out)
		}(jobItem, in, out, wg)
		in = out
	}

}

func goDataSignerCrc32(in, out chan string) {
	data := <-in

	defer goDebuger.DebugTimeStamp(time.Now(), 2, "goDataSignerCrc32", data)

	out <- DataSignerCrc32(data)
}

// we can only run a DataSignerCrc32 function one at the given moment
// need to wait while a DataSignerCrc32 function is calculating
// simaphor = 1 open -> can start DataSignerCrc32
// simaphor = 0 open -> can't start DataSignerCrc32
var simaphorGoDataSignerMd5 uint32 = 1

func goDataSignerMd5(data string) string {
	// goDebuger.DebugTimeStamp(2, "goDataSignerMd5", data)

	for {
		if swapped := atomic.CompareAndSwapUint32(&simaphorGoDataSignerMd5, 1, 0); swapped {
			result := DataSignerMd5(data)
			atomic.SwapUint32(&simaphorGoDataSignerMd5, 1)
			return result
		}
		runtime.Gosched()
	}
}

// SingleHash Calculate
func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for input := range in {
		wg.Add(1)
		go func(wg *sync.WaitGroup, in interface{}, out chan interface{}) {
			defer wg.Done()
			data := fmt.Sprintf("%v", in)

			defer goDebuger.DebugTimeStamp(time.Now(), 1, "SingleHash", data)

			md5Data := goDataSignerMd5(data)

			// crc32Md5Data := DataSignerCrc32(md5Data)
			crc32Md5DataIn := make(chan string)
			crc32Md5DataOut := make(chan string)
			go goDataSignerCrc32(crc32Md5DataIn, crc32Md5DataOut)
			crc32Md5DataIn <- md5Data

			// crc32Data := DataSignerCrc32(data)
			crc32DataIn := make(chan string)
			crc32DataOut := make(chan string)
			go goDataSignerCrc32(crc32DataIn, crc32DataOut)
			crc32DataIn <- data

			out <- <-crc32DataOut + "~" + <-crc32Md5DataOut

		}(wg, input, out)
	}
}

type outChannel struct {
	i      int
	result string
}

// MultiHash Calculate
func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	for input := range in {
		wg.Add(1)
		go func(wg *sync.WaitGroup, in interface{}, out chan interface{}) {
			defer wg.Done()
			data := fmt.Sprintf("%v", in)

			channel := make(chan outChannel)

			// Run calculation of CRC32 five times
			for th := 0; th < 6; th++ {
				go func(th int, data string, out chan outChannel) {
					result := DataSignerCrc32(strconv.Itoa(th) + data)
					out <- outChannel{th, result}
				}(th, data, channel)
			}

			// Collecting results from 5 runned DataSignerCrc32
			results := make(map[int]string)
			for i := 0; i < 6; i++ {
				c := <-channel
				results[c.i] = c.result
			}
			close(channel)

			// Combine results
			result := ""
			for i := 0; i < 6; i++ {
				result = result + results[i]
			}

			out <- result
		}(wg, input, out)
	}
}

// CombineResults combine given string from the channel. Using "_" as a separator.
func CombineResults(in, out chan interface{}) {

	// get all results from the input channel
	var r []string
	for result := range in {
		r = append(r, fmt.Sprintf("%v", result))
	}

	// sort
	sort.Slice(r, func(i, j int) bool {
		return r[i] < r[j]
	})

	// combine to the final string
	result := r[0]
	for i := 1; i < len(r); i++ {
		result = result + "_" + r[i]
	}
	out <- result
}

func main() {

	defer goDebuger.DebugTimeStamp(time.Now(), 0, "Main")

	// inputData := []int{0, 1}
	inputData := []int{0, 1, 1, 2, 3, 5, 8}

	testResult := ""

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			dataRaw := <-in
			data, ok := dataRaw.(string)
			if !ok {
				fmt.Println("cant convert result data to string")
			}
			testResult = data
		}),
	}

	ExecutePipeline(hashSignJobs...)

	fmt.Println("")
	fmt.Println("Result =", testResult)
}
