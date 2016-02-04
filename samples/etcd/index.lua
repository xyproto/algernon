local j = JNode()
j:set("value", "etcd works")

print("Send:")
print(tostring(j))

local status = j:PUT("http://127.0.0.1:2379/v2/keys/mykey")

log(status)

local k = JNode()
local status = k:GET("http://127.0.0.1:2379/v2/keys/mykey")

log(status)

print("Receive:")
print(tostring(k))


