# what is this?

this is the client side for [EmmyLua](https://github.com/EmmyLua) and supported to debug the [gopher-lua](https://github.com/yuin/gopher-lua)'s lua code

# how to use

1. install EmmyLua for goland/idea/... and learn how to debug lua with this plugin
2. go get github.com/edolphin-ydf/gopherlua-debugger
3. edit your go.mod, add this line `replace github.com/yuin/gopher-lua => github.com/edolphin-ydf/gopher-lua v0.0.0-20191105142246-92ca436742b9`
4. in your go code, after `L := lua.NewState()` add a new line `lua_debugger.Preload(L)`, of course, you should import gopher-lua-debugger
5. in your lua code, anywhere you want to start debug/break, add the following line
```lua
local dbg = require('emmy_core')
dbg.tcpConnect('localhost', 9966)
```

# why need replace gopher-lua?

the original gopher-lua doesn't implement the `debug.hook()` func. the replacement implement it and fix a bug for debug.getlocal().
if the author accepted my patch, the replacement won't need anymore. But you need the replace now!!

# what is `lua_debugger.Preload(L)` do?

this will preload the emmy_core module which support the `tcpConnect`, then you can connect to the EmmyLua server to start debug

# limitation

the EmmyLua provide two ways to start a debug, the ide as a server and the ide as a client.
but the gopherlua-debugger only support ide as server, lua instance as client.

# contribution

issue and pr are welcome