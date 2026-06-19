# VHS tape for the monkeytui demo GIF.
# Regenerate with:  vhs demo.tape   (needs vhs + ttyd + ffmpeg)
# Uses a fixed -seed so the typed words match the target for a clean run.

Output demo.gif

Set Shell bash
Set FontSize 18
Set Width 1100
Set Height 640
Set Padding 22
Set TypingSpeed 90ms
Set Theme "GruvboxDark"

Hide
Type "./monkeytui -mode words -words 18 -seed 42"
Enter
Sleep 1s
Show

Sleep 800ms
Type "well why same interest out but or how education account it real night late book simply mile large"
Sleep 2s
Sleep 3s
