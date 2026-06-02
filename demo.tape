# Copyright 2026 Optiqor contributors
# SPDX-License-Identifier: Apache-2.0
#
# vhs tape — records the kerno doctor MVP demo for the README hero GIF.
#
# IMPORTANT: vhs records a non-interactive shell that cannot type a sudo
# password. The Makefile runs vhs itself as root so the recorded shell
# is already privileged — no `sudo` typed in this tape.
#
# Run with:    make demo
# Direct:      sudo $(which vhs) demo.tape
# Output:      demo.gif (~5 MB target after gifsicle)

Output demo.gif

# ── visual config ──
Set Shell      "bash"
Set FontSize   14
Set Width      1100
Set Height     720
Set Theme      "Dracula"
Set TypingSpeed 35ms
Set Padding    20

# ── silent setup ──
# Hide the prompt-cleanup, PATH wiring, and demo-config selection so
# the viewer sees `kerno doctor` — the clean marketing command — and
# nothing else. KERNO_CONFIG points at the test-tuned thresholds that
# fire on synthetic chaos load (production defaults are calibrated for
# real incidents and won't trip on a healthy SSD dev box).
Hide
Type "export PS1='$ '"
Enter
Type "export PATH=$(pwd)/bin:$PATH"
Enter
Type "export KERNO_CONFIG=$(pwd)/scripts/verify-config.yaml"
Enter
Type "clear"
Enter
Show

# ── intro ──
Type "# Kerno doctor — kernel-level incident diagnosis in one command"
Sleep 1s
Enter
Sleep 500ms

Type "# Step 1: induce a synthetic disk-fsync bottleneck"
Sleep 600ms
Enter
Type "kerno chaos --induce disk-sat --duration 25s --intensity high --yes &"
Enter
Sleep 2s

Hide
Type "clear"
Enter
Show

# ── doctor run — the hero shot ──
Type "# Step 2: ask the kernel what's wrong"
Sleep 600ms
Enter
Type "kerno doctor --duration 10s"
Enter

# Doctor takes ~10s to collect + a couple seconds to render the
# box-drawn report. Wait for output to fully paint.
Sleep 15s

# ── outro ──
Type "# Root cause identified. Same flow runs in CI: 'make verify'."
Sleep 3s
