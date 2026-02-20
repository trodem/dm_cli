package tools

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"cli/internal/systeminfo"
	"cli/internal/ui"
)

func RunSystemAuto() int {
	return RunSystem(nil)
}

func RunSystem(_ *bufio.Reader) int {
	s := systeminfo.Collect()

	ui.PrintSection("System Snapshot")
	ui.PrintKV("Generated", s.GeneratedAt.Format(time.RFC3339))
	ui.PrintKV("Host", valueOrDash(s.System.Hostname))
	ui.PrintKV("OS", fmt.Sprintf("%s/%s", s.System.OS, s.System.Arch))
	ui.PrintKV("CPU", fmt.Sprintf("%d", s.System.CPUCount))
	if !s.System.BootTime.IsZero() {
		uptime := time.Since(s.System.BootTime).Round(time.Minute)
		ui.PrintKV("Boot time", s.System.BootTime.Format(time.RFC3339))
		ui.PrintKV("Uptime", uptime.String())
	}
	if s.Memory.TotalBytes > 0 {
		used := s.Memory.TotalBytes - s.Memory.FreeBytes
		ui.PrintKV("Memory", fmt.Sprintf("%s used / %s total", formatBytes(used), formatBytes(s.Memory.TotalBytes)))
	}

	ui.PrintSection("Disks")
	if len(s.Disks) == 0 {
		fmt.Println(ui.Muted("- none"))
	} else {
		fmt.Printf("%-5s %-13s %-13s %-6s\n", "Name", "Used", "Total", "Use%")
		for _, d := range s.Disks {
			used := d.SizeBytes - d.FreeBytes
			usedPct := 0.0
			if d.SizeBytes > 0 {
				usedPct = (float64(used) / float64(d.SizeBytes)) * 100
			}
			fmt.Printf("%-5s %-13s %-13s %5.1f%%\n", d.Name, formatBytes(used), formatBytes(d.SizeBytes), usedPct)
		}
	}

	ui.PrintSection("Interfaces")
	if len(s.Interfaces) == 0 {
		fmt.Println(ui.Muted("- none"))
	} else {
		fmt.Printf("%-30s %-6s %-17s %s\n", "Name", "State", "MAC", "Addresses")
		for _, inf := range s.Interfaces {
			state := "down"
			if inf.Up {
				state = "up"
			}
			addrs := "-"
			if len(inf.Addresses) > 0 {
				addrs = strings.Join(inf.Addresses, ", ")
			}
			fmt.Printf("%-30s %-6s %-17s %s\n", inf.Name, state, valueOrDash(inf.Hardware), addrs)
		}
	}

	ui.PrintSection("Wi-Fi")
	ui.PrintKV("Connected", valueOrDash(s.ConnectedWiFi))
	if len(s.WiFiNetworks) == 0 {
		fmt.Println(ui.Muted("- no networks detected"))
	} else {
		fmt.Printf("%-32s %-8s %s\n", "SSID", "Signal", "Auth")
		for _, net := range s.WiFiNetworks {
			fmt.Printf("%-32s %-8s %s\n", valueOrDash(net.SSID), valueOrDash(net.Signal), valueOrDash(net.Authentication))
		}
	}

	ui.PrintSection("LAN Neighbors (ARP)")
	if len(s.LANNeighbors) == 0 {
		fmt.Println(ui.Muted("- none"))
	} else {
		limit := len(s.LANNeighbors)
		if limit > 25 {
			limit = 25
		}
		fmt.Printf("%-16s %-17s %s\n", "IP", "MAC", "Type")
		for i := 0; i < limit; i++ {
			n := s.LANNeighbors[i]
			fmt.Printf("%-16s %-17s %s\n", n.IP, n.MAC, n.Type)
		}
		if len(s.LANNeighbors) > limit {
			fmt.Println(ui.Muted(fmt.Sprintf("... and %d more", len(s.LANNeighbors)-limit)))
		}
	}

	if len(s.Warnings) > 0 {
		ui.PrintSection("Warnings")
		for _, w := range s.Warnings {
			fmt.Printf("- %s\n", ui.Warn(w))
		}
	}
	return 0
}

func formatBytes(n uint64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
		tb = 1024 * gb
	)
	switch {
	case n >= tb:
		return fmt.Sprintf("%.2fTB", float64(n)/float64(tb))
	case n >= gb:
		return fmt.Sprintf("%.2fGB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.2fMB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.2fKB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

func valueOrDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}
