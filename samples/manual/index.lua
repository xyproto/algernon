aprint([[

$css = "]] .. file2url("css/style.css") .. [["
$user = "]] .. Username() .. [["

doctype 5
html
    head
        title Amber
        link[href=$css][rel="stylesheet"][type="text/css"]
    body
        h1 Hi, #{$user}
        div#content
            p
                | This text goes into the first paragraph.
                |  As well as this text. And this. Yarr.

            a[href="/"][onClick="history.go(-1)"] Back
]])
