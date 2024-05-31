-- Function to read and render *.md files in the script directory
local function render_markdown_files()
    local md_files = readglob("*.md")
    for i = 1, #md_files do
        local content = md_files[i]
        if content then
            print('<div class="formula-box">')
            mprint(content)
            print('</div>')
        else
            print('<div class="formula-box"><p>Error: Unable to read markdown file</p></div>')
        end
    end
end

-- Main function
local function main()
    print[[
    <!doctype html>
    <html>
    <head>
        <title>Math Formulas</title>
        <link rel="stylesheet" type="text/css" href="style.css">
    </head>
    <body>
        <div class="container">
            <h1>Mathematical Formulas</h1>
            <div class="section">
    ]]

    -- Render markdown files
    render_markdown_files()

    print[[
            </div>
        </div>
    </body>
    </html>
    ]]
end

-- Execute main function
main()
