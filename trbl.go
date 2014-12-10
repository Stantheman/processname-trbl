package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		logp("Usage: %v <pid>", os.Args[0])
		os.Exit(1)
	}
	pid := os.Args[1]

	start, stop, err := get_stack_positions(pid)
	if err != nil {
		logp("%s\n", err)
		os.Exit(2)
	}

	cmdline, err := get_process_name(pid, start, stop)
	if err != nil {
		logp("%s\n", err)
		os.Exit(3)
	}

	fmt.Println(cmdline)
}

func get_stack_positions(pid string) (uint64, uint64, error) {

	logp("Getting position of the stack from /proc/%v/maps", pid)

	fh, err := os.Open("/proc/" + pid + "/maps")
	if err != nil {
		return 0, 0, fmt.Errorf("/proc/%v/maps can't be accessed: %v\n", pid, err)
	}
	defer fh.Close()
	reader := bufio.NewReader(fh)

	// grab the line with the stack info
	stack_line := ""
	for line, err := reader.ReadString('\n'); ; line, err = reader.ReadString('\n') {
		if err != nil {
			return 0, 0, err
		}
		if line[len(line)-8:] == "[stack]\n" {
			stack_line = line
			break
		}
	}
	if stack_line == "" {
		return 0, 0, fmt.Errorf("couldn't find the stack!")
	}

	// get the hex addrs out
	hyphen := strings.IndexByte(stack_line, '-')
	space := strings.IndexByte(stack_line, ' ')
	if hyphen == -1 || space == -1 {
		return 0, 0, fmt.Errorf("couldn't parse the map file")
	}

	start_hex := stack_line[:hyphen]
	end_hex := stack_line[hyphen+1 : space]

	// convert hex to dec
	start_dec, err := strconv.ParseUint(start_hex, 16, 64)
	if err != nil {
		return 0, 0, err
	}
	end_dec, err := strconv.ParseUint(end_hex, 16, 64)
	if err != nil {
		return 0, 0, err
	}

	logp("Found the hex range %s, converted to decimal range %v-%v", stack_line[:space], start_dec, end_dec)

	return start_dec, end_dec, nil
}

// get_process_name takes advantage of elf binary loading getting process information
// stuffed at the bottom of the stack at process invocation
func get_process_name(pid string, start, stop uint64) (string, error) {

	logp("Grabbing the stack from /proc/%s/mem, from %v up to %v", pid, start, stop)

	fh, err := os.Open("/proc/" + pid + "/mem")
	if err != nil {
		return "", err
	}
	defer fh.Close()

	size := stop - start
	buf := make([]byte, size)
	if n, err := fh.ReadAt(buf, int64(start)); n != int(size) || err != nil {
		return "", fmt.Errorf("returned %v bytes, expected %v. Error: %v", n, size, err)
	}

	logp("Got the stack, it's %v bytes long. Starting to read from the end", len(buf))
	logp("Skipping the last nine bytes since they're nulls")

	// last 9 bytes are null because the bottom of the stack is backed up by sizeof void*, plus ending null of interp string
	// TODO: should uname -m bits to decide between 8 and 4 sizeof void *
	buf = buf[:len(buf)-9]

	// filename is first, take everything after the last null
	interp_idx := bytes.LastIndex(buf, []byte{'\x00'})
	interp := buf[interp_idx+1:]
	logp("Got the interpreter, it's %s. Moving onto the environment", interp)

	// env is next. get the size of environ to go backwards that much
	env_size, err := get_env_offset(pid)
	if err != nil {
		return "", err
	}
	logp("Environment is %v bytes long", env_size)
	logp("Getting the invocation by going backwards until two nulls happen in a row")

	// finally the invocation. the invoc is null separated, look until you see two nulls in a row
	final_index, last_seen := 0, byte('\x01')
	for i := interp_idx - env_size + 1; i > 0; i-- {
		if buf[i] == '\x00' {
			if last_seen == '\x00' {
				final_index = i + 2
				break
			}
			last_seen = '\x00'
			continue
		}
		last_seen = buf[i]
	}

	invoc := buf[final_index : interp_idx-env_size]
	invoc = bytes.Replace(invoc, []byte{'\x00'}, []byte{' '}, -1)
	logp("The invocation is %v bytes long. Replacing nulls with spaces", len(invoc))

	return string(invoc), nil
}

// get_env_offset returns the size of the environment so we know how much to skip up the stack
func get_env_offset(pid string) (int, error) {
	logp("Getting the length of the environment of pid %v by checking /proc/%v/environ", pid, pid)
	fh, err := os.Open("/proc/" + pid + "/environ")
	if err != nil {
		return 0, err
	}
	defer fh.Close()

	// wc appears to just try to read until it gets to the end, so try that
	buf := make([]byte, 16384)
	n, err := fh.Read(buf)
	if err != nil {
		return 0, err
	}

	return n, nil
}

func logp(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "- "+format+" \n\n", a...)
}
