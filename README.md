# Running instructions
- Build both servers. From the project root (where `README.md` is located):
- Build the HTTP server:
    - `cd cmd/http_server`
    - `go build .`
- Build the TCP server:
    - `cd cmd/tcp_server`
    - `go build main.go`
- Ensure MySQL DB and Redis is running
- Run the TCP server first
- Run the HTTP server next (forms TCP connection pool on startup)

# Additional flags
Both the HTTP and TCP servers support logging/monitoring configuration using
flags.
- See HTTP server flags: `./http_server -h`
    - Example: `./http_server --logLevel=DEBUG --logOutput=FILE`
    

- See TCP server flags: `./tcp_server -h`
    - Example: `./main --cpuprofile=cpu.prof --logLevel=ERROR --logOutput=ALL`

# Enabling monitoring & visualisation
Optionally, start Prometheus and Grafana:
- Ensure prometheus and grafana are installed (available on brew)
- Start Prometheus:
  - In `/tools/prometheus`: `prometheus --config.file=promtheus.yml`
  - Supported metrics:
    - `namespace_subsystem_login_count`
- Start Grafana:
  - TODO

# Project Structure
Follows https://github.com/golang-standards/project-layout
- `api` stores protocol definition used in TCP client-server communication
- `cmd` stores main source code for HTTP and TCP servers
- `configs` (gitignored) stores password files
- `internal` stores helper code used by `cmd`
- `test` stores profiling/benchmarking/testing code
