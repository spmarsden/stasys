#! /usr/bin/env fish

###############################################################################
# Author: Stephen P Marsden
# Date: 2022-09-06
#
# Description: Outputs a single line summary of key CPU, RAM, and Swap details, 
#              intended to be embedded in the GNOME topbar.
#
# Dependencies: fish
#               ip
#               sensors
#               vmstat
#               free
#
###############################################################################


function rx_tx_Mb
    # Sums the RX and TX bytes over all interfaces.
    # Returns: <unix time [ns]> <RX [Mb]> <TX [Mb]>
    
    # Record the timestamp.
    set -l time_ns (date '+%S%N')

    # Get the ip ouput for all interfaces
    set -l ip_output (ip -s link)
    # Split by line.
    set ip_output (string split "\n" $ip_output)
    # Compact whitespace so elements are only separated by one space. 
    for i in (seq 1 (count $ip_output))
        set ip_output[$i] (string split " " $ip_output[$i] | string join -n " ")
    end
    
    # Sum the RX and TX bytes.
    set -l rx 0
    set -l tx 0
    for i in (seq (count $ip_output))
        set -l line (string split " " $ip_output[$i])
        if [ "$line[1]" = "RX:" ]
            set rx (math $rx + (echo $ip_output[(math $i + 1)] | string split " ")[1])
        else if [ "$line[1]" = "TX:" ]
            set tx (math $tx + (echo $ip_output[(math $i + 1)] | string split " ")[1])
        end
    end
    
    # Convert from bytes to bits
    set rx (math $rx "*" 8)
    set tx (math $tx "*" 8)
    
    # Convert from b to Mb
    # RX
    set rx (echo "$rx / 100000" | bc)
    if test (count (string split "" $rx) -eq 1)
        set rx "0$rx"
    end
    set rx (string sub $rx -s 1 -e -1).(string sub $rx -s -1 -l 1)
    # TX
    set tx (echo "$tx / 100000" | bc)
    if test (count (string split "" $tx) -eq 1)
        set tx "0$tx"
    end
    set tx (string sub $tx -s 1 -e -1).(string sub $tx -s -1 -l 1)
    
    # Output
    echo $time_ns $rx $tx
end


## Network Usage 1/2 ##########################################################

# Record the time and cumulative network usage.
#   1: unix timestamp in ns.
#   2: RX in Mb
#   3: TX in Mb
set -l t_rx_tx_start (rx_tx_Mb | string split " ")


## CPU Usage ##################################################################

# Average the CPU idle time over a second.
# Gets a multiline output.
# Takes the last line.
# Splits it by space. This results in a list with multiple empty elements.
# Joins the list, ignoring the empty elements, placing a single space between each.
# Split by spaces again. This time there won't be any empty elements.
# Take the 15th element which contains the idle time.
set -l cpu_idle (vmstat 1 1 | tail -1 | string split " " | string join -n " " | string split " ")[15]

# Invert to get the utilisation.
set -l cpu_usage (echo "100 - $cpu_idle" | bc)


## Network Usage 2/2 ##########################################################

# Record the time and cumulative network usage.
#   1: unix timestamp in ns.
#   2: RX in Mb
#   3: TX in Mb
set -l t_rx_tx_end (rx_tx_Mb | string split " ")

# Calculate the change.
# Time
set -l delta_t_ns (math $t_rx_tx_end[1] - $t_rx_tx_start[1])
# RX
set -l delta_rx_Mb (math $t_rx_tx_end[2] - $t_rx_tx_start[2])
# TX
set -l delta_tx_Mb (math $t_rx_tx_end[3] - $t_rx_tx_start[3])

# Calculate rate.
set -l drx_dt_Mbps (echo "scale=1; 1000000000 * $delta_rx_Mb / $delta_t_ns" | bc)
set -l dtx_dt_Mbps (echo "scale=1; 1000000000 * $delta_tx_Mb / $delta_t_ns" | bc)

# Add in missing leading/trailing zeros.
if [ (string sub "$drx_dt_Mbps" -s 1 -e 1) = "." ]
    set drx_dt_Mbps "0$drx_dt_Mbps"
else if [ "$drx_dt_Mbps" = "0" ]
    set drx_dt_Mbps "$drx_dt_Mbps.0"
end
if [ (string sub "$dtx_dt_Mbps" -s 1 -e 1) = "." ]
    set dtx_dt_Mbps "0$dtx_dt_Mbps"
else if [ "$dtx_dt_Mbps" = "0" ]
    set dtx_dt_Mbps "$dtx_dt_Mbps.0"
end
# Add the units.
set drx_dt_Mbps "$drx_dt_Mbps Mb/s"
set dtx_dt_Mbps "$dtx_dt_Mbps Mb/s"


## CPU Frequency ##############################################################

# Get the frequency of all of the cores. The content will look like:
#   1: cpu
#   2: MHz		:
#   3: XXXX.XXX
#   4: cpu
#   5: MHz      :
#   6: XXXX.XXX
#   ...
set -l cpu_freqs (cat /proc/cpuinfo | grep "cpu MHz" | string split " ")

# Calculate the average frequency.
# Caclulate the total.
set -l cpu_freqs_average 0
set -l n_cpu (math (count $cpu_freqs) / 3)
for i_cpu in (seq 1 $n_cpu)
    set -l index (math $i_cpu "*" 3)
    set cpu_freqs_average (math $cpu_freqs_average + $cpu_freqs[$index])
end
# Take the average.
set cpu_freqs_average (echo "$cpu_freqs_average / $n_cpu" | bc)
# Format the string.
if test $cpu_freqs_average -ge 1000
    set cpu_freqs_average (echo "scale=2; $cpu_freqs_average / 1000" | bc) GHz
else if test $cpu_freqs_average -lt 1
    set cpu_freqs_average (echo "$cpu_freqs_average * 1000" | bc) kHz
else
    set cpu_freqs_average (echo "$cpu_freqs_average / 1" | bc) MHz
end


## CPU Temperature ############################################################

# Use sensors instead. This is slower, but compatible with more systems.
set -l cpu_temps (sensors | grep Core | string split " " | string join -n " " | string split " ")
set -l cpu_temp 0
set -l n_cpu (math (count $cpu_temps) / 9)
for i_cpu in (seq 1 $n_cpu)
    set -l index (math (math 9 "*" $i_cpu) - 6)
    set cpu_temp (math $cpu_temp + (string sub $cpu_temps[$index] -s 2 -e -2))
end
set cpu_temp (echo "$cpu_temp / $n_cpu" | bc)


## RAM Usage ##################################################################

# Read in the RAM free and used.
set -l ram_info (free -m | grep Mem | string split " " | string join -n " " | string split " ")
set -l ram_usage
# Calculate the usage as a percentage.
set ram_usage (echo (echo "100 * $ram_info[3]" | bc) / $ram_info[2] | bc)


## SWAP Usage #################################################################

# Read in the Swap free and used.
set -l swap_info (free -m | grep Swap | string split " " | string join -n " " | string split " ")
set -l swap_usage
# Calculate the usage as a percentage.
# But first, check that swap space is set up.
if test "$swap_info[2]" -ne 0 
    set swap_usage (echo (echo "100 * $swap_info[3]" | bc) / $swap_info[2] | bc)
end


## Output #####################################################################

set -l output "CPU:   $cpu_freqs_average   $cpu_usage%   $cpu_temp°C"
set output "$output   |   RAM:   $ram_usage%"
if test -n "$swap_usage"
    set output "$output   |   Swap: $swap_usage%"
end
set output "$output   |   ↑$dtx_dt_Mbps   ↓$drx_dt_Mbps"

echo $output
