-- Store an IP address given as URL/?ip=123.123.123.123
-- Serve a document with the last given IP if no new value is given.

local urldataTable = urldata()

if urldataTable["ip"] ~= nil then

    local IP = urldataTable["ip"]

    -- opens a file in write + create mode
    file = io.open(scriptdir("ip.md"), "w+")

    -- sets the default output file as test.lua
    io.output(file)

    --- header
    io.write("<!-- theme: dark -->\n")
    io.write("# GetIP 1.0\n")

    -- appends a word test to the last line of the file
    io.write("Last IP received = `" .. IP.."`\n")

    -- closes the open file
    io.close(file)

    -- serve the newly generated markdown document
    serve("ip.md")

else

    -- check if an existing ip.md file can be read
    file = io.open(scriptdir("ip.md"), "r")
    if file ~= nil then

        -- serve the existing markdown document
        serve("ip.md")
    else

        -- usage information for the user
        print("Pass an IP address with ?ip=...")
    end
end
