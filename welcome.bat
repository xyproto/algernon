@echo off
echo Try editing the markdown file in the "samples" directory and see the
echo results instantly in the browser at http://localhost:3000/
.\algernon.exe --dev --conf serverconf.lua --dir samples --httponly --debug --autorefresh --bolt --server
