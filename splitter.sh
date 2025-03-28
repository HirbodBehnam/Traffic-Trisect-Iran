#!/bin/bash
split -b 100M -d --additional-suffix=.7z "$1" "$2"
