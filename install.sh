#! /bin/bash

###############################################################################
#
# Must be run from the package root directory.
#
# Compiles the code and links it to into the bin directory.
#
###############################################################################

if [ "$EUID" -ne 0 ]
then 
    echo "Must be run as root."
    exit 1
fi
# Build the timer.
echo "Building stasys."
go build -o bin/ src/stasys.go

echo "Linking executables."
# Check that a binary directory can be found.
if [ -d "../../bin" ]
then
    ln -sf "$PWD/bin/stasys" "../../bin/"
else 
    echo "No bin directory found."
fi