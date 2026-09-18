// Command mockclient drives a dashpot invocation the way a real MCP
// client would: spawns it, sends ten identical tools/call requests in
// rapid succession, then prints a one-line summary to its own stdout.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

type request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: mockclient <dashpot-command> [args...]")
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	reader := bufio.NewReader(stdout)
	send(stdin, 1, "initialize", nil)
	readLine(reader)

	forwarded, tripped := 0, 0
	for i := range 10 {
		send(stdin, i+2, "tools/call", map[string]any{
			"name":      "bash",
			"arguments": map[string]any{"command": "cargo test"},
		})
		if strings.Contains(readLine(reader), "CIRCUIT BREAKER") {
			tripped++
		} else {
			forwarded++
		}
	}

	send(stdin, 99, "test/invocationCount", nil)
	countLine := readLine(reader)

	_ = stdin.Close()
	_ = cmd.Wait()

	fmt.Printf("forwarded=%d tripped=%d %s\n", forwarded, tripped, countLine)
}

func send(w io.Writer, id int, method string, params any) {
	b, err := json.Marshal(request{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		log.Fatal(err)
	}
}

func readLine(r *bufio.Reader) string {
	line, err := r.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return strings.TrimRight(line, "\n")
}
