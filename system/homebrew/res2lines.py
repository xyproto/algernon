#!/usr/bin/env python
# -*- coding: utf-8 -*-
#
# Format the output from homebrew-go-resources
#
# Alexander F Rødseth <xyproto@archlinux.org>, 14-10-2015
#

def main():
    data = open("resources.txt").read()
    rubyline = ""
    for line in [x.strip() for x in data.split('\n')]:
        if line.startswith("go_resource"):
            rubyline = line.split()[1].split('"')[1]
        elif line.startswith(":revision"):
            rubyline += " " + line.split()[2].split('"')[1]
            print(rubyline)

if __name__ == "__main__":
    main()
