# cleansrt

Download youtube subtitles with `yt-dlp`, I use this program to edit the outputted `.srt` files to be human readable.

```
yt-dlp --write-auto-sub --sub-lang en --skip-download --convert-subs srt -o "transcript.%(ext)s" "<youtube_url>"

cleansrt -o transcript.txt transcript.en.srt 
```
