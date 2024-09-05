## Three ways of displaying a UNIX Day counter

* Client side only (index.html with JavaScript).
* REST (index.html with JavaScript that fetches data from the /day endpoint, which is served by day/index.lua).
* Server side only (index.po2 Pongo2 template together with a data.lua file that makes functions or data available for the template).

## NOTE

The REST example should be started within the `rest` folder so that Algernon serves `day/index.lua` as `/day` and not as `/rest/day`.
