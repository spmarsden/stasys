# stasys

A script for generating a one line summary of the system status.

## Dependencies

* go
* ip
* sensors
* vmstat
* free

## Installation

* Recommended installation location: `/opt/` or `/usr/local/opt`
* To compile the binary and link it into `/bin/` or `/usr/local/bin`:
    * `sudo ./install.sh`

## Usage

`stasys`

## Example Output 

`CPU: 1.3 GHz  46%  51°C   |   RAM: 24%   |   ↑0.6 Mb/s  ↓31.0 Mb/s`