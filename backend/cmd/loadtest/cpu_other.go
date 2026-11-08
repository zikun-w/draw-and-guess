//go:build !linux

package main

// Linux reports process CPU time through getrusage. Other hosts return zero.
func cpuSeconds() float64 { return 0 }
