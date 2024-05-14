#!/usr/bin/env perl

use strict;
use warnings;

my $backend_name = shift @ARGV;  # the name of the backend to match

# Set flag to start/stop printing lines
my $print_lines = 0;

# Read from STDIN
while (my $line = <STDIN>) {
    chomp $line;

    # Check if the line defines a new backend.
    if ($line =~ /^backend\s+/i) {
        # If it's the target backend, set the flag to start printing.
        if ($line =~ /$backend_name/i) {
            $print_lines = 1;
        } else {
            # If it's a new backend and we were already printing,
            # stop.
            last if $print_lines;
        }
    }

    # Print the line if within the desired backend block.
    print "$line\n" if $print_lines;
}
