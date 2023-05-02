package workers

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
	"go.uber.org/zap"
	"model-hub/helper"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type WorkerInfo struct {
	ID                WorkerId
	ElapsedTimeString string
	CPUPercent        string
	RAMInMB           string
}

func (wm *WorkerManager) logResourceUsage() {
	updateIntervalStr := helper.GetEnv("METRICS_DISPLAY_FREQUENCY", "30")
	updateSecondsInterval, err := strconv.Atoi(updateIntervalStr)
	if err != nil {
		panic("METRICS_DISPLAY_FREQUENCY has invalid value: " + updateIntervalStr)
	}
	interval := time.Duration(updateSecondsInterval) * time.Second
	for {
		startTime := time.Now()
		cmd := exec.Command(
			"nvidia-smi",
			"--query-gpu=memory.used,memory.total",
			"--format=csv,noheader,nounits",
		)
		output, err := cmd.Output()

		gpuPercent := 0.0
		if err == nil {
			var gpuMemoryUsed, gpuMemoryTotal uint64
			_, _ = fmt.Sscanf(string(output), "%d, %d", &gpuMemoryUsed, &gpuMemoryTotal)
			gpuPercent = (float64(gpuMemoryUsed) / float64(gpuMemoryTotal)) * 100
		}

		v, _ := mem.VirtualMemory()
		totalRAM := float64(v.Total) / (1024 * 1024)
		availableRAM := float64(v.Available) / (1024 * 1024)

		percentages, _ := cpu.Percent(time.Duration(1000)*time.Millisecond, true)

		var cpuInfoBuilder strings.Builder
		for i, percentage := range percentages {
			cpuInfoBuilder.WriteString(fmt.Sprintf("Core%d: %.2f%%|", i, percentage))
		}

		cpuInfo := strings.TrimSuffix(cpuInfoBuilder.String(), " | ")

		var workerInfos []WorkerInfo
		var maxIDLen, maxElapsedLen, maxCPULen, maxRAMLen int

		for _, worker := range wm.workers {
			if !worker.IsLaunched() {
				continue
			}
			p, err := process.NewProcess(int32(worker.cmd.Process.Pid))
			if err != nil {
				wm.logger.Error("Failed to get process", zap.String("workerId", string(worker.ID)), zap.Error(err))
				continue
			}
			cpuPercent, err := p.CPUPercent()
			if err != nil {
				wm.logger.Error("Failed to get CPU usage", zap.String("workerId", string(worker.ID)), zap.Error(err))
			}

			memInfo, err := p.MemoryInfo()
			if err != nil {
				wm.logger.Error("Failed to get memory usage", zap.String("workerId", string(worker.ID)), zap.Error(err))
			}

			ramInMB := float64(memInfo.RSS) / (1024 * 1024)

			idLen := len(worker.ID)
			elapsedLen := len(worker.ElapsedTimeString())
			cpuLen := len(fmt.Sprintf("%.2f", cpuPercent))
			ramLen := len(fmt.Sprintf("%.2f", ramInMB))

			if idLen > maxIDLen {
				maxIDLen = idLen
			}
			if elapsedLen > maxElapsedLen {
				maxElapsedLen = elapsedLen
			}
			if cpuLen > maxCPULen {
				maxCPULen = cpuLen
			}
			if ramLen > maxRAMLen {
				maxRAMLen = ramLen
			}

			workerInfos = append(workerInfos, WorkerInfo{
				ID:                worker.ID,
				ElapsedTimeString: worker.ElapsedTimeString(),
				CPUPercent:        fmt.Sprintf("%.2f", cpuPercent),
				RAMInMB:           fmt.Sprintf("%.2f", ramInMB),
			})
		}
		var formattedWorkerInfo []string
		for _, info := range workerInfos {
			formattedWorkerInfo = append(formattedWorkerInfo, fmt.Sprintf("⚙️ Worker %s%-*s (⏱️lifetime: %s%-*s): 🖥️ CPU: %s%-*s%% | 💾 RAM: %s%-*s MB",
				info.ID, maxIDLen-len(info.ID), "", info.ElapsedTimeString, maxElapsedLen-len(info.ElapsedTimeString), "", info.CPUPercent, maxCPULen-len(info.CPUPercent), "", info.RAMInMB, maxRAMLen-len(info.RAMInMB), ""))

		}
		fmt.Printf("====== 🎮 TOTAL GPU USAGE:  %.2f%% =======\n", gpuPercent)
		fmt.Printf("====== 💾 TOTAL RAM: %.2f MB | AVAILABLE RAM: %.2f MB =======\n", totalRAM, availableRAM)
		fmt.Printf("====== 🖥️ CPU USAGE (%s)=======\n", cpuInfo)
		fmt.Printf("====== 🤖 WORKER INFO =======\n%s\n", strings.Join(formattedWorkerInfo, "\n"))
		fmt.Printf("====== ⏱️ TIME TAKEN FOR METRICS: %.2f s =======\n", time.Since(startTime).Seconds())
		time.Sleep(interval)
	}
}
func (w *Worker) ElapsedTimeString() string {
	elapsedTime := time.Since(w.startTime)
	hours := int(elapsedTime.Hours())
	minutes := int(elapsedTime.Minutes()) % 60
	seconds := int(elapsedTime.Seconds()) % 60

	timeString := ""
	if hours > 0 {
		timeString = fmt.Sprintf("%d hours ", hours)
	}
	if minutes > 0 {
		timeString += fmt.Sprintf("%d minutes ", minutes)
	}
	timeString += fmt.Sprintf("%d seconds", seconds)
	return timeString
}
