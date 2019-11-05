local dbg = require('emmy_core')
dbg.tcpConnect('localhost', 9966)
--package.cpath = package.cpath .. ';/Users/edolphin/Library/Application Support/JetBrains/Toolbox/apps/Goland/ch-0/192.6817.25/GoLand.app.plugins/intellij-emmylua/classes/debugger/emmy/mac/?.dylib'
--local dbg = require('emmy_core')
--dbg.tcpConnect('localhost', 9966)


local t = {aa = 1, bb = 2}
for k, v in pairs(t) do
    print(k, v)
end

local w = 10
local s = "asdf"
local s1 = "qwer"
local s2 = s .. s1
local bo = true
local bo1 = false
function test(x)
    local y = 3
    local z = 5
    local v = y + z + w
end

local a = 1
local b = 2
local c = a+b
test(c)
