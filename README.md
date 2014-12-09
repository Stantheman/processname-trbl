terrible process getter
====

this gets the information contained in /proc/pid/cmdline, which is what ps uses when it prints a filename.

When a process is loaded into memory for execution, the filename, environment, and argv are saved at the bottom new process's stack.

/proc gives you a read-only copy of the process's data, and a map that tells you where the stack is :). 

example stderr
--------
 - Getting position of the stack from /proc/8342/maps
 
 - Found the hex range ffd02000-ffd23000, converted to decimal range 4291829760-4291964928
 
 - Grabbing the stack from /proc/8342/mem, from 4291829760 up to 4291964928
 
 - Got the stack, it's 135168 bytes long. Starting to read from the end
 
 - Skipping the last nine bytes since they're nulls
 
 - Got the interpreter, it's /usr/bin/tail. Moving onto the environment
 
 - Getting the length of the environment of pid 8342 by checking /proc/8342/environ
 
 - Environment is 2955 bytes long
 
 - Getting the invocation by going backwards until two nulls happen in a row
 
 - The invocation is 35 bytes long. Replacing nulls with spaces

other stuff
----

Some files don't have stack entries, but only have thread stack entries. Need to investigate that more

The kernel source specifies the offset from the bottom of the stack to essentially be (the bottom - sizeof(void\*)). I don't know why they subtract the space for the pointer. Maybe it's used in a special marking way somewhere else?

It would be neat to parse /proc/kcore somehow (get the offsets? is it possible?)

I don't believe this works for a.out files, need to verify/check the kernel source
