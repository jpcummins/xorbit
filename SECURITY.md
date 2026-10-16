# Security

`xorbit` handles sensitive entropy and a 24-word BIP-39 mnemonic code.
Treat both as secrets.

## Use it safely

- Run it offline on a fresh TailsOS session, in a private terminal. Do not
  use `tmux`, `screen`, `script`, serial-console logging, cameras, or other
  recording devices.
- Review the source and verify the build you run. The program accepts no
  arguments and requires its controlling terminal.
- Keep the XOR screen private. It displays the dice digest, kernel RNG bytes,
  and final entropy; these values are for verification and need not be saved.
- Write the mnemonic code on paper when it is revealed. It is shown once.
  The verification step echoes the words you type, so keep that screen
  private too. Do not copy entropy or mnemonic words into a shell, file, or
  another program.
- Store the paper securely. Shut down TailsOS after use; do not suspend or
  hibernate the computer that displayed the mnemonic code.

## Protections and limits

The program hardens its Linux process, silences standard output and error,
disables core dumps, locks and wipes sensitive buffers, and does not log,
write data files, open network connections, or start child processes. The
runtime may retain copies of sensitive data until shutdown; TailsOS's no-swap,
RAM-wiping environment is part of the security assumptions. The program
scans the controlling terminal before every input read (including each dice
entry), before displaying the entropy mix, and before revealing the mnemonic
code. These scans are snapshots, not an exclusive lock: another process can
open the pty after a scan and close it before the next one, including while a
line is being entered. They narrow but do not eliminate that race, and do not
make an untrusted computer safe.
If a process's fd table cannot be inspected, the scan checks its controlling
terminal instead. This avoids rejecting unrelated protected desktop processes,
but may not detect a process that opened the pty without making it its
controlling terminal.

The operator must trust the hardware, Linux kernel, source code, and build.
A compromised kernel or root account can defeat these protections. Physical
observation, recording devices, and a compromised terminal emulator are also
outside the program's protection.

## Reporting a vulnerability

Please report vulnerabilities privately through GitHub's private
vulnerability reporting feature, if it is enabled for this repository. Do
not publish unpatched vulnerability details in a public issue. Never include
real dice rolls, entropy values, or mnemonic words in a report; use synthetic
data or the project's deterministic test fixtures.
