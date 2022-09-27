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

func t_rx_tx_Mb() (int64, float32, float32) {

	// Record the time that the data was read in.
	read_time := time.Now().Unix()

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
	rx_Mb := float32(rx) * 8 / 1e6
	tx_Mb := float32(tx) * 8 / 1e6

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

	// Record the initial cumulative data use.
	t, rx, tx := t_rx_tx_Mb()

	fmt.Println(t, rx, tx)

	freq_MHz := cpu_freq_MHz()

	fmt.Println(freq_MHz)
}
