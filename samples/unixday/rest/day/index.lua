local function current_unix_day()
    return math.floor(os.time(os.date("!*t")) / 86400)
end

local function get_seconds_to_next_update_check()
    local now = os.time(os.date("!*t"))
    local nextDayStart = (current_unix_day() + 1) * 86400
    return nextDayStart - now
end

local unixDay = current_unix_day()
local secondsToNextUpdate = get_seconds_to_next_update_check()

local responseData = {
    unixday = unixDay,
    secondsToNextUpdate = secondsToNextUpdate
}

content("application/json")
print(json(responseData))
