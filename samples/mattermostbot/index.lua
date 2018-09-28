-- Can talk back and forth with Mattermost, given a Mattermost web hook

-- Fill in these with your own data:
local webhook = "http://mattermost/hooks/asdf123"
local avatar = "http://imageserver/bender.png"
local botname = "Bender9000"

content("text/html; charset=utf-8")

-- Only accept POST requests
if method() ~= "POST" then
    print("Unsupported method: " .. method())
    return
end

local fields = urldata(body())
local username = fields["user_name"]
local command = fields["text"]:sub(fields["trigger_word"]:len() + 2)

log(username .. "> " .. command)

j = JNode()
j:set("text", "Hi " .. username .. "! You said: " .. command)
j:set("icon_url", avatar)
j:set("username", botname)
local status = j:send(webhook)

log(status)
