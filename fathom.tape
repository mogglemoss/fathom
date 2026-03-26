# fathom demo — NOAA tides TUI
# Run with: vhs fathom.tape

Output assets/fathom.gif

Set Shell "fish"
Set FontSize 14
Set FontFamily "JetBrainsMono Nerd Font"
Set Width 1200
Set Height 700
Set Theme "Dark+"
Set PlaybackSpeed 1.0
Set TypingSpeed 80ms

# Launch fathom with Boston station
Hide
Type "/Users/scott/go/bin/fathom --station 8443970"
Enter
Sleep 5s
Show

# Switch to Almanac
Type "2"
Sleep 2s

# Scroll through the almanac
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# Switch to Station view
Type "3"
Sleep 2s

# Back to tide view
Type "1"
Sleep 1s

# Navigate to next day
Right
Sleep 1.5s

# Navigate back to today
Type "t"
Sleep 1.5s

# Open date input overlay
Type "d"
Sleep 800ms
Type "apr 4"
Sleep 500ms
Enter
Sleep 1.5s

# Go back to today
Type "t"
Sleep 1s

# Toggle full help
Type "?"
Sleep 1.5s
Type "?"
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
