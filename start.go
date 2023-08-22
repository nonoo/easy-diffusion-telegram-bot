package main

import (
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

const easyDiffusionStartTimeout = 20 * time.Second
const easyDiffusionPingInterval = 500 * time.Millisecond

func startEasyDiffusionIfNeeded() {
	out, err := exec.Command("pgrep", "uvicorn").Output()
	if err == nil && len(out) > 0 {
		fmt.Println("easy-diffusion is already running")
	} else {
		fmt.Println("starting easy-diffusion... ")
		cmd := exec.Cmd{
			Path: params.EasyDiffusionPath,
		}
		if err := cmd.Start(); err != nil {
			panic(fmt.Sprint("can't start easy diffusion: ", err))
		}
	}

	fmt.Println("checking easy-diffusion...")
	startedAt := time.Now()
	var lastPingAt time.Time
	for {
		elapsedSinceLastPing := time.Since(lastPingAt)
		if elapsedSinceLastPing < easyDiffusionPingInterval {
			time.Sleep(easyDiffusionPingInterval - elapsedSinceLastPing)
		}

		res, err := req.Ping()
		if err != nil {
			if err == syscall.ECONNREFUSED {
				panic(fmt.Sprint("can't start easy diffusion: ", err))
			}
		}
		if res {
			break
		}

		if time.Since(startedAt) > easyDiffusionStartTimeout {
			panic("can't start easy diffusion: ping timeout")
		}

		lastPingAt = time.Now()
		fmt.Println("  ping...")
	}
	fmt.Println("  ok")
}
