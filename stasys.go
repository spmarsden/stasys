package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func stdout2fields(stdout bytes.Buffer) [][]string {
	// Convert a byte stream from a command line programme into a 2D slice
	// split first by \n, then by whitespace.

	// Split by \n
	stdout_lines := strings.Split(stdout.String(), "\n")

	// Split into fields.
	var fields [][]string
	for _, line := range stdout_lines {
		fields = append(fields, strings.Fields(line))
	}

	return fields
}

func t_rx_tx_Mb() (float64, float64, float64) {

	// Record the time that the data was read in.
	read_time := float64(time.Now().UnixNano()) / 1e9

	// Run ip to get the received / transmitted bits.
	// Define the command.
	cmd := exec.Command("ip", "-s", "link")
	// Add a stream for the stdout.
	var ip_stdout bytes.Buffer
	cmd.Stdout = &ip_stdout
	// Run the command.
	_ = cmd.Run()
	// 4 ms.

	// Split the output into lines.
	ip_lines := strings.Split(ip_stdout.String(), "\n")
	// Storage for the current interface name.
	interface_name := ""
	// Storage for if we want to do something with the line that follows.
	is_rx := false
	is_tx := false
	// Storage for the cumulative RX and TX vaues.
	rx := 0
	tx := 0
	for _, line := range ip_lines {
		// Split each line into fields.
		ip_fields := strings.Fields(line)
		// Check if the line is an interface heading.
		if len(ip_fields) > 2 && ip_fields[2][0:1] == "<" {
			// Strip formatting away from interface name.
			interface_name = strings.Split(ip_fields[2], ",")[0][1:]
		} else if interface_name == "LOOPBACK" {
			// Skip the LOOPBACK interface.
			continue
		} else if is_rx {
			// Read the RX bits for this interface.
			drx, _ := strconv.Atoi(ip_fields[0])
			// Add the bits to the running total.
			rx += drx
			// Reset the flag ready for the next line.
			is_rx = false
		} else if is_tx {
			// Read the TX bits for this interface.
			dtx, _ := strconv.Atoi(ip_fields[0])
			// Add the bits to the running total.
			tx += dtx
			// Reset the flag ready for the next line.
			is_tx = false
		} else if len(ip_fields) > 0 {
			// Check if the next line is RX.
			if ip_fields[0] == "RX:" {
				is_rx = true
			}
			// Check if the next line is TX.
			if ip_fields[0] == "TX:" {
				is_tx = true
			}
		}
	}

	// Convert from B to Mb.
	rx_Mb := float64(rx) * 8 / 1e6
	tx_Mb := float64(tx) * 8 / 1e6

	return read_time, rx_Mb, tx_Mb
}

func cpu_freq_MHz() float64 {
	// Storage for the average frequency.
	average_frequency := 0.
	// Record how many CPUs were measured.
	n_cpu := 0.
	// Read in the file containing the frequencies.
	data, _ := os.ReadFile("/proc/cpuinfo")
	// Record the CPU frequencies.
	for _, line := range strings.Split(string(data), "\n") {
		// Split the line by whitespace.
		fields := strings.Fields(line)
		// If the line contains CPU information, record it.
		if len(fields) > 3 && fields[0] == "cpu" && fields[1] == "MHz" {
			// Add the frequency to the total.
			dfreq, _ := strconv.ParseFloat(fields[3], 64)
			average_frequency += dfreq
			// Record that another CPU has been read.
			n_cpu += 1
		}
	}
	// Calculate the average frequency.
	average_frequency /= n_cpu
	// Return that average.
	return average_frequency
}

func main() {

	// General /////////////////////////////////////////////////////////////////

	// Get the sensors output. This will be used for temperatures and fan speed.
	// Define the command.
	cmd_sensors := exec.Command("sensors")
	// Add a stream for the stdout.
	var sensors_stdout bytes.Buffer
	cmd_sensors.Stdout = &sensors_stdout
	// Run the command.
	_ = cmd_sensors.Run() // 14 ms.

	// Split the output.
	sensors := stdout2fields(sensors_stdout)

	// Network Usage 1/2 ///////////////////////////////////////////////////////

	// Record the time and cumulative network usage.
	network_t_start_s, network_rx_start_Mb, network_tx_start_Mb := t_rx_tx_Mb()

	// CPU /////////////////////////////////////////////////////////////////////

	// CPU Usage.
	// Run vmstat to get the idle percentage.
	// Define the command.
	cmd_vmstat := exec.Command("vmstat", "1", "1")
	// Add a stream for the stdout.
	var vmstat_stdout bytes.Buffer
	cmd_vmstat.Stdout = &vmstat_stdout
	// Run the command.
	_ = cmd_vmstat.Run() // 6 ms.

	// Split output into lines.
	vmstat_lines := strings.Split(vmstat_stdout.String(), "\n")
	// Find the index of the idle percentage.
	id_index := -1
	for i, heading := range strings.Fields(vmstat_lines[1]) {
		if heading == "id" {
			id_index = i
			break
		}
	}
	// Get the data divided into fields.
	vmstat_data := strings.Fields(vmstat_lines[2])
	// Get the idle percentage.
	id_percentage, _ := strconv.Atoi(vmstat_data[id_index])
	// Get the complement of the idle percentage.
	cpu_percentage := 100 - id_percentage

	// CPU Frequency.
	// Get the average CPU frequency.
	freq_MHz := cpu_freq_MHz()
	// Format the output string.
	cpu_freq_str := ""
	if freq_MHz > 1e3 {
		cpu_freq_str = fmt.Sprintf("%.1f GHz", freq_MHz/1e3)
	} else {
		cpu_freq_str = fmt.Sprintf("%.0f MHz", freq_MHz)
	}

	// CPU Temperature.
	cpu_temp := 0.
	// Count the number of cores being averaged over.
	n_cpu := 0
	// Get the sum of the core temperatures.
	for _, line := range sensors {
		if len(line) > 0 && line[0] == "Core" {
			temp := line[2][1 : len(line[2])-3]
			dt, _ := strconv.ParseFloat(temp, 64)
			cpu_temp += dt
			n_cpu++
		}
	}
	cpu_temp /= float64(n_cpu)

	// Fan Speed ///////////////////////////////////////////////////////////////
	// for _, line := range sensors {
	// 	if len(line) > 0 && line[0] ==
	// }

	// Memory Usage ////////////////////////////////////////////////////////////

	// Run free to get the memory stats.
	// Define the command.
	cmd_free := exec.Command("free", "-m")
	// Add a stream for the stdout.
	var free_stdout bytes.Buffer
	cmd_free.Stdout = &free_stdout
	// Run the command.
	_ = cmd_free.Run() // 6 ms.

	// Split the output.
	free := stdout2fields(free_stdout)

	// Find the index of the total and free columns.
	used_index := -1
	total_index := -1
	for i, entry := range free[0] {
		if entry == "total" {
			total_index = i + 1
		} else if entry == "used" {
			used_index = i + 1
		}
		if used_index > -1 && total_index > -1 {
			break
		}
	}

	// Find the index of the memory entry.
	mem_index := -1
	swap_index := -1
	for i, line := range free {
		if len(line) == 0 {
			continue
		}
		if line[0] == "Mem:" {
			mem_index = i
		} else if line[0] == "Swap:" {
			swap_index = i
		}
	}

	mem_used, _ := strconv.ParseFloat(free[mem_index][used_index], 64)
	mem_total, _ := strconv.ParseFloat(free[mem_index][total_index], 64)
	mem_percentage := 0.
	if mem_total > 0 {
		mem_percentage = 100. * mem_used / mem_total
	}

	swap_percentage := -1.
	if swap_index > -1 {
		swap_used, _ := strconv.ParseFloat(free[swap_index][used_index], 64)
		swap_total, _ := strconv.ParseFloat(free[swap_index][total_index], 64)
		if swap_total > 0 {
			swap_percentage = 100. * swap_used / swap_total
		}
	}

	// Network Usage 2/2 ///////////////////////////////////////////////////////

	// Record the time and cumulative network usage.
	network_t_stop_s, network_rx_stop_Mb, network_tx_stop_Mb := t_rx_tx_Mb()

	// Calculate the transfer rate.
	network_t_delta_s := network_t_stop_s - network_t_start_s
	network_rx_delta_Mb := network_rx_stop_Mb - network_rx_start_Mb
	network_tx_delta_Mb := network_tx_stop_Mb - network_tx_start_Mb
	drx_dt := network_rx_delta_Mb / network_t_delta_s
	dtx_dt := network_tx_delta_Mb / network_t_delta_s

	// Output //////////////////////////////////////////////////////////////////

	divider := " | "

	output := ""
	output += fmt.Sprintf("CPU: %s %d%% %.0f°C",
		cpu_freq_str,
		cpu_percentage,
		cpu_temp)
	output += divider
	output += fmt.Sprintf("RAM: %.0f%%", mem_percentage)
	if swap_percentage > -1 {
		output += divider
		output += fmt.Sprintf("Swap: %.0f%%", swap_percentage)
	}
	output += divider
	output += fmt.Sprintf("↑%.1f Mb/s ↓%.1f Mb/s", dtx_dt, drx_dt)

	fmt.Println(output)

}
