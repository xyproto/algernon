content("text/html")

local page = HTML5("On the fly")

page:metaCharset("UTF-8")
    :addStyle([[
        body  { font-family: sans-serif; margin: 2em; max-width: 40em }
        h1    { color: #336699 }
        h2    { border-bottom: 1px solid #ccc; padding-bottom: 0.2em; margin-top: 1.6em }
        .lead { font-style: italic; color: #555 }
        .admin{ background: #fff5cc; padding: 1em; border-left: 4px solid #c90 }
        code  { background: #f0f0f0; padding: 0 0.3em }
        dt    { font-weight: bold; margin-top: 0.4em }
    ]])

local body = page:tag("body")

body:addNewTag("h1"):addContent("Server side facts on the fly")

local dl = body:addNewTag("dl")

local function fact(label, value)
    dl:addNewTag("dt"):addContent(label)
    dl:addNewTag("dd"):addNewTag("code"):addContent(value)
end

fact("Server clock", os.date())
fact("HTTP method",  method())
fact("URL path",     urlpath())

print(page:html())
