package cgroups

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func NewCpu(root string) *cpuController {
	return &cpuController{
		root: filepath.Join(root, string(Cpu)),
	}
}

type cpuController struct {
	root string
}

func (c *cpuController) Name() Name {
	return Cpu
}

func (c *cpuController) Path(path string) string {
	return filepath.Join(c.root, path)
}

func (c *cpuController) Create(path string, resources *specs.Resources) error {
	if err := os.MkdirAll(c.Path(path), defaultDirPerm); err != nil {
		return err
	}
	if cpu := resources.CPU; cpu != nil {
		for _, t := range []struct {
			name  string
			value *uint64
		}{
			{
				name:  "rt_period_us",
				value: cpu.RealtimePeriod,
			},
			{
				name:  "rt_runtime_us",
				value: cpu.RealtimeRuntime,
			},
			{
				name:  "shares",
				value: cpu.Shares,
			},
			{
				name:  "cfs_period_us",
				value: cpu.Period,
			},
			{
				name:  "cfs_quota_us",
				value: cpu.Quota,
			},
		} {
			if t.value != nil {
				if err := ioutil.WriteFile(
					filepath.Join(c.Path(path), fmt.Sprintf("cpu.%s", t.name)),
					[]byte(strconv.FormatUint(*t.value, 10)),
					defaultFilePerm,
				); err != nil {
					return err
				}
			}
		}

	}
	return nil
}

func (c *cpuController) Update(path string, resources *specs.Resources) error {
	return c.Create(path, resources)
}

func (c *cpuController) Stat(path string, stats *Stats) error {
	f, err := os.Open(filepath.Join(c.Path(path), "cpu.stat"))
	if err != nil {
		return err
	}
	defer f.Close()
	// get or create the cpu field because cpuacct can also set values on this struct
	stats.cpuMu.Lock()
	cpu := stats.Cpu
	if cpu == nil {
		cpu = &CpuStat{}
		stats.Cpu = cpu
	}
	stats.cpuMu.Unlock()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if err := sc.Err(); err != nil {
			return err
		}
		key, v, err := parseKV(sc.Text())
		if err != nil {
			return err
		}
		switch key {
		case "nr_periods":
			cpu.Throttling.Periods = v
		case "nr_throttled":
			cpu.Throttling.ThrottledPeriods = v
		case "throttled_time":
			cpu.Throttling.ThrottledTime = v
		}
	}
	return nil
}
