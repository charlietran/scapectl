#!/bin/sh
# Render icons/states/sheet.png: every shipped set at 36px on light and dark
# menubar backgrounds. Run after gen.py install.
set -e
cd "$(dirname "$0")"
src=../../internal/tray/icons
rows=""
for set in ghost outline dim; do
  for theme in light dark; do
    if [ $theme = light ]; then bg='#e8e8ea'; fg=black; label='#333'; else bg='#2a2a2c'; fg=white; label='#ddd'; fi
    magick -background "$bg" -fill "$label" -font /System/Library/Fonts/Helvetica.ttc -pointsize 14 \
      \( -size 120x48 -gravity West label:"$set / $theme" \) \
      \( $src/$set/nodongle_$fg.png -resize 36x36 \) \
      \( $src/$set/disconnected_$fg.png -resize 36x36 \) \
      \( $src/$set/connected_$fg.png -resize 36x36 \) \
      \( $src/$set/muted_$fg.png -resize 36x36 \) \
      -gravity Center +smush 28 -bordercolor "$bg" -border 16x6 row_${set}_$theme.png
    rows="$rows row_${set}_$theme.png"
  done
done
magick -background '#e8e8ea' -fill '#333' -font /System/Library/Fonts/Helvetica.ttc -pointsize 12 \
  \( -size 152x24 xc:"#e8e8ea" \) \
  \( -size 64x24 -gravity Center label:"no dongle" \) \
  \( -size 64x24 -gravity Center label:"off" \) \
  \( -size 64x24 -gravity Center label:"connected" \) \
  \( -size 64x24 -gravity Center label:"muted" \) \
  +append header.png
magick header.png $rows -append sheet.png
rm -f row_*.png header.png
echo "wrote sheet.png"
