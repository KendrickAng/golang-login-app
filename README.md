Entry task for Shared Services backend internship

# Project Structure
Follows https://github.com/golang-standards/project-layout
- `api` stores protocol definition used in TCP client-server communication
- `cmd` stores main source code for HTTP and TCP servers
- `configs` (gitignored) stores password files
- `internal` stores helper code used by `cmd`
- `test` stores profiling/benchmarking/testing code