#!/bin/sh
for x in react.production.min react-dom.production.min react.development react-dom.development; do
  echo "Downloading $x.js..."
  curl --progress-bar -L -o "/tmp/$x.js" "https://unpkg.com/umd-react/dist/$x.js" -C - && mv -f "/tmp/$x.js" "$x.js" || exit 1
done
echo Done.
