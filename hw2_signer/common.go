package main

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"
)

type job func(in, out chan interface{})

const (
	MaxInputDataLen = 100
)

var (
	dataSignerOverheat uint32 = 0
	DataSignerSalt            = ""
)

var OverheatLock = func() {
	// DEBUG Start
	debug := false
	if debug {
		start := time.Now()
		fmt.Printf("\tOverheatLock() is started\n")
		defer func(start time.Time) {
			d := time.Since(start)
			end := time.Now()
			re := regexp.MustCompile(`m=.*`)
			fmt.Printf("\tOverheatLock() start %v end %v duraction %v \n", re.FindString(start.String()), re.FindString(end.String()), d)
		}(start)
	}
	// DEBUG Finish
	for {
		if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 0, 1); !swapped {
			fmt.Println("OverheatLock happend")
			time.Sleep(time.Second)
		} else {
			break
		}
	}
}

var OverheatUnlock = func() {
	for {
		if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 1, 0); !swapped {
			fmt.Println("OverheatUnlock happend")
			time.Sleep(time.Second)
		} else {
			break
		}
	}
}

var DataSignerMd5 = func(data string) string {
	// DEBUG Start
	debug := false
	if debug {
		start := time.Now()
		fmt.Printf("\t\tDataSignerMd5(%v) is started\n", data)
		defer func(start time.Time, data string) {
			d := time.Since(start)
			end := time.Now()
			re := regexp.MustCompile(`m=.*`)
			fmt.Printf("\t\tDataSignerMd5(%v) start %v end %v duraction %v \n", data, re.FindString(start.String()), re.FindString(end.String()), d)
		}(start, data)
	}
	// DEBUG Finish

	OverheatLock()
	defer OverheatUnlock()

	data += DataSignerSalt
	dataHash := fmt.Sprintf("%x", md5.Sum([]byte(data)))
	time.Sleep(10 * time.Millisecond)
	return dataHash
}

var DataSignerCrc32 = func(data string) string {
	data += DataSignerSalt
	crcH := crc32.ChecksumIEEE([]byte(data))
	dataHash := strconv.FormatUint(uint64(crcH), 10)
	time.Sleep(time.Second)
	return dataHash
}
