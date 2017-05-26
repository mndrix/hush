#!/usr/bin/env perl
use strict;
use warnings;
use autodie qw( open );
use IPC::Open2;
use Term::ReadKey;


# does "git credential-cache" have what we need?
my ($r, $w);
open2($r, $w, 'git', 'credential-cache', 'get');
print $w "protocol=hush\nhost=local\n\n";
close($w);
while (<$r>) {
    chomp;
    if (s/^password=//g) {
        print;
        exit;
    }
}

# ask user for his password
print STDERR "askpass: $ARGV[0]: ";
ReadMode('noecho');
my $password = <STDIN>;
ReadMode('normal');
chomp $password;
print $password;  # output password for hush to read


# store password in git-credential-cache
open my $git, '|-', 'git', 'credential-cache', 'store';
print $git "protocol=hush\nhost=local\nusername=michael\npassword=$password\n\n";
