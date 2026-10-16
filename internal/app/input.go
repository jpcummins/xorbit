package application

// Operator input handling: readLine scans the controlling terminal before
// every TTY read, fills the locked Workspace.Line buffer, and returns the
// trimmed line. Every caller wipes WS.Line when it is done with the line.

// readLine reads one operator line into Workspace.Line and returns it
// trimmed of surrounding spaces and tabs.
func (a *Runner) readLine() ([]byte, error) {
	if err := a.CheckTerminal(); err != nil {
		wipe(a.WS.Line)
		return nil, err
	}
	n, err := a.Term.ReadLine(a.WS.Line)
	if err != nil {
		wipe(a.WS.Line)
		return nil, err
	}
	return trimSpace(a.WS.Line[:n]), nil
}

func trimSpace(b []byte) []byte {
	for len(b) > 0 && isSpace(b[0]) {
		b = b[1:]
	}
	for len(b) > 0 && isSpace(b[len(b)-1]) {
		b = b[:len(b)-1]
	}
	return b
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t'
}

// letter returns the lowercase letter if line contains exactly one ASCII
// letter, and 0 otherwise. Commands are case-insensitive.
func letter(line []byte) byte {
	if len(line) != 1 {
		return 0
	}
	c := line[0]
	if 'A' <= c && c <= 'Z' {
		c += 'a' - 'A'
	}
	if 'a' <= c && c <= 'z' {
		return c
	}
	return 0
}
