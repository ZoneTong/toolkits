package main

import (
	"github.com/shirou/gopsutil/mem"
)

func GetSysMem() (total, free uint64, err error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return
	}

	total = v.Total
	free = v.Free
	return
}
