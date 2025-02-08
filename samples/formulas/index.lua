-- Function to read and render *.md files in the script directory
local function render_markdown_files()
    local md_files = readglob("*.md")
    local additionalJS = ""
    for i = 1, #md_files do
        local content = md_files[i]
        if content then
            print('<div class="formula-box">')
            additionalJS = mprint_ret(content)
            print('</div>')
        else
            print('<div class="formula-box"><p>Error: Unable to read markdown file</p></div>')
        end
    end
    return additionalJS
end

-- Main function
local function main()
    print[[
    <!doctype html>
    <html>
    <head>
        <title>Mathematical Formulas</title>
        <link rel="stylesheet" type="text/css" href="style.css">
    </head>
    <body>
        <div class="container">
            <h1>Mathematical Formulas</h1>
            <div class="section">
    ]]

    -- Render markdown files
    additionalScriptTag = render_markdown_files()

    print[[
            </div>
        </div>
    ]]

    if additionalScriptTag ~= "" then
      print(additionalScriptTag)
    end

    print[[
    </body>
    </html>
    ]]
end

-- Execute main function
main()
