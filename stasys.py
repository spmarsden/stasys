#! /usr/bin/env python3

###############################################################################
# Author: Stephen P Marsden
# Date: 2022-09-06
#
# Description: Outputs a single line summary of key CPU, RAM, and Swap details, 
#              intended to be embedded in the GNOME topbar.
#
# Dependencies: python3
#               numpy
#               ip
#               sensors
#               vmstat
#               free
#
###############################################################################

# For gettinf the current unix time.
from time import time
# For executing shell commands.
from subprocess import Popen, PIPE, DEVNULL
# for quicker array calculations.
from numpy import array

def t_rx_tx_Mb():
    """
    Retreives the current recorded received and transmitted bits, along with the
    unix time at which there were accessed. Subsequent calls can be used to 
    calculate the rate of data transfer.
    
    Args:
        None
    
    Returns:
        float - unix timestamp [s]
        int   - received bits [Mb]
        int   - transmitted bits [Mb]
    
    Raises:
        None
    
    ToDo:
        None
    """

    # Get the unix timestamp.
    unix_time = time()

    # Run ip to get the received / transmitted bits.
    process = Popen(['ip', '-s','link'], 
                           stdout=PIPE, 
                           stderr=DEVNULL,
                           universal_newlines=True)
    # Read the output.
    stdout = [line for line in process.stdout.readlines()]

    # Remove the loopback interface.
    # Fins the start of the loopback.
    skip_start = None
    for i,line in enumerate(stdout):
        if line[0] != " ":
            if "LOOPBACK" in line:
                skip_start = i
                break
    # Find the end of the loopback.
    skip_end = None
    for i,line in enumerate(stdout[skip_start+1:]):
        if line[0] != " ":
            skip_end = skip_start + i
            break
    # Remove the loopback entry.
    stdout = stdout[0:skip_start] + stdout[skip_end+1:]
    
    # Split each line by whitespace.
    stdout = [line.split() for line in stdout]

    # Get the received/transmitted line numbers.
    rx_lines = [i+1 for i in range(len(stdout)) if stdout[i][0]=="RX:"]
    tx_lines = [i+1 for i in range(len(stdout)) if stdout[i][0]=="TX:"]

    # Sum the received/transmitted bits and convert to Mb.
    rx_sum_Mb = sum([int(stdout[i][0]) for i in rx_lines]) * 8 / 1e6
    tx_sum_Mb = sum([int(stdout[i][0]) for i in tx_lines]) * 8 / 1e6

    return array([unix_time, rx_sum_Mb, tx_sum_Mb])

def cpu_freq_MHz():
    """
    Reads in the current CPU frequencies and calculates the average, returning
    a single frequency.
    
    Args:
        None
    
    Returns:
        float - Frequency in MHz.
    
    Raises:
        None
    
    ToDo:
        None
    """

    # Storage for all the CPU frequencies.
    frequencies_MHz = []
    # Read in the file containing the values.
    with open("/proc/cpuinfo","r") as cpuinfo:
        # Find each line containing a CPU frequency.
        for line in cpuinfo.readlines():
            if line.startswith("cpu MHz"):
                # Record the frequency.
                frequencies_MHz.append(float(line.split()[-1]))

    # Get the average frequency.
    return sum(frequencies_MHz) / len(frequencies_MHz)

def main():

    ## General #################################################################

    # Get the sensors output. This will be used for temperatures and fan speed.
    sensors_process = Popen(['sensors'],
                            stdout=PIPE,
                            stderr=DEVNULL,
                            universal_newlines=True)
    # Read in the output from sensors.
    sensors_stdout = [line.split() for line in sensors_process.stdout.readlines()]
    

    ## Network Usage 1/2 #######################################################

    # Record the time and cumulative network usage.
    #   0: unix timestamp.
    #   1: RX in Mb
    #   2: TX in Mb
    t_rx_tx_start = t_rx_tx_Mb()


    ## CPU #####################################################################

    ### CPU Usage.
    # Run vmstat to get the idle percentage.
    vmstat_process = Popen(['vmstat', '1','1'], 
                           stdout=PIPE, 
                           stderr=DEVNULL,
                           universal_newlines=True)
    # Read the output.
    vmstat_stdout = [line.split() for line in vmstat_process.stdout.readlines()]
    # Find the index of the idle percentage.
    id_index = vmstat_stdout[1].index("id")
    # Get the inverse of the idle percentage.
    cpu_percentage = 100 - int(vmstat_stdout[2][id_index])

    ### CPU Frequency.
    # Get the average CPU frequency.
    freq_MHz = cpu_freq_MHz()
    # Format the output string.
    cpu_freq_str = ""
    if freq_MHz > 1e3:
        cpu_freq_str = f"{freq_MHz/1e3:.1f} GHz"
    else:
        cpu_freq_str = f"{round(freq_MHz)} MHz"

    ### CPU Temperature.
    cpu_temps = [line[2] for line in sensors_stdout if len(line)>0 and line[0]=="Core"]
    # Take the average temperature.
    cpu_temps_average = round(sum([float(value[1:-2]) for value in cpu_temps]) / len(cpu_temps))
    
    
    ## Memory Usage ############################################################

    ### Gather data.
    # Run free to gather memory information.
    free_process = Popen(['free', '-m'], 
                           stdout=PIPE, 
                           stderr=DEVNULL,
                           universal_newlines=True)
    # Read the output.
    free_stdout = [line.split() for line in free_process.stdout.readlines()]
    # Add an empty entry at the start of the title row to keep everything in line.
    free_stdout[0] = [''] + free_stdout[0]

    ### Get the column indexes.
    # Total capacity.
    total_index = free_stdout[0].index("total")
    # Capacity used.
    used_index = free_stdout[0].index("used")

    ### RAM
    # Find the RAM index.
    ram_index = [line[0] for line in free_stdout].index("Mem:")
    # Get the RAM values.
    ram_total = int(free_stdout[ram_index][total_index])
    ram_used  = int(free_stdout[ram_index][used_index ])
    # Get the RAM used percentage.
    ram_percentage = round(100 * ram_used/ram_total)

    ### Swap
    swap_index = [line[0] for line in free_stdout].index("Swap:")
    if swap_index < len(free_stdout):
        # Get the swap values.
        swap_total = int(free_stdout[swap_index][total_index])
        swap_used  = int(free_stdout[swap_index][used_index ])
        # Get the RAM used percentage.
        if swap_total > 0:
            swap_percentage = round(100 * swap_used/swap_total)

    ## Network Usage 2/2 #######################################################

    # Record the time and cumulative network usage.
    #   0: unix timestamp.
    #   1: RX in Mb
    #   2: TX in Mb
    t_rx_tx_stop = t_rx_tx_Mb()

    # Calculate the transfer rate.
    delta_t_rx_tx = t_rx_tx_stop - t_rx_tx_start
    drx_dt = f"{delta_t_rx_tx[1] / delta_t_rx_tx[0]:.1f}"
    dtx_dt = f"{delta_t_rx_tx[2] / delta_t_rx_tx[0]:.1f}"

    ## Output ##################################################################

    # Define the separator.
    separator = "   |   "

    # Put the output together.
    output = ""
    output += f"CPU:   {cpu_freq_str}   {cpu_percentage}%   {cpu_temps_average}°C"
    output += separator
    output += f"RAM: {ram_percentage}%"
    if swap_total > 0:
        output += separator
        output += f"Swap: {swap_percentage}%"
    output += separator
    output += f"↑{dtx_dt} Mb/s   ↓{drx_dt} Mb/s"


    print(output)
    return

if __name__=="__main__":
    main()