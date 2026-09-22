Output demo/session-manager.gif

Set Theme "Dracula"
Set FontSize 16
Set Framerate 12
Set Margin 0
Set PlaybackSpeed 1

Env COLORTERM truecolor
Env TERM xterm-256color

Type `PATH="$PWD/demo/mock-bin:$PATH" ./claude-cleaner --claude-dir /tmp/claude-demo`
Enter
Sleep 1500ms

# Skip update prompt if present; harmless on list screen.
Type "n"
Sleep 300ms

# --- Project Detail ---
# Enter with no project selected opens the current project.
Enter
Sleep 900ms

# Lock the project to demonstrate protection.
Type "l"
Sleep 500ms

# Unlock again so session actions become available.
Type "l"
Sleep 500ms

# Select two individual conversation sessions.
Space
Sleep 300ms
Down
Space
Sleep 300ms

# Open per-session delete confirmation.
Enter
Sleep 700ms

# Cancel: this demo never deletes seeded data.
Escape
Sleep 500ms

# Clear session selection and return to project list.
Type "n"
Sleep 250ms
Escape
Sleep 1000ms

# --- Forget Project ---
# Show the explicit forget-project confirmation.
Type "X"
Sleep 800ms

# Cancel: source and Claude demo data remain untouched.
Escape
Sleep 500ms

Type "q"
