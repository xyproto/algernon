-- Function to read and render *.md files in the script directory
local function render_markdown_files()
    local md_files = readglob("*.md")
    local additional_script_tag = ""
    for i = 1, #md_files do
        local content = md_files[i]
        if content then
            print('<div class="formula-box">')
            additional_script_tag = mprint_ret(content)
            print('</div>')
        else
            print('<div class="formula-box"><p>Error: Unable to read markdown file</p></div>')
        end
    end
    return additional_script_tag
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
    additional_script_tag = render_markdown_files()

    print[[
            </div>
        </div>
    ]]

    if additional_script_tag ~= "" then
        print(additional_script_tag)
    end

    print[[
    </body>
    </html>
    ]]
end

-- Execute main function
main()
