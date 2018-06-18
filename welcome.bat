@echo off
echo Try editing the markdown file in samples/welcome and see the
echo results instantly in the browser at http://localhost:3000/
.\algernon.exe --dev --conf serverconf.lua --dir algernon-1.10\samples --httponly --debug --autorefresh --bolt --server
