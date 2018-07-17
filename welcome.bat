@echo off
echo Try editing the markdown file in samples/welcome and see the
echo results instantly in the browser at http://localhost:3000/
.\algernon.exe --dev --conf serverconf.lua --dir samples/welcome --httponly --debug --autorefresh --bolt --server
