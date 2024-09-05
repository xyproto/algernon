function current_unix_day()
    local secondsPerDay = 86400
    return math.floor(unixnano() / 1e9 / secondsPerDay)
end
