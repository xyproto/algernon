#!/bin/sh
for x in react react-dom; do
  for variant in production.min development; do
    file="$x.$variant.js"
    echo "Downloading $file..."
    curl --progress-bar -L -o "/tmp/$file" "https://unpkg.com/$x/umd/$file" -C - && mv -f "/tmp/$file" "$file" || exit 1
  done
done
echo Done.
