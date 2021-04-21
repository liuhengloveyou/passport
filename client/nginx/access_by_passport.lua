local ck = require "resty.cookie"
local cookie, err = ck:new()
if not cookie then
    ngx.log(ngx.ERR, err)
    return
end

local field, err = cookie:get("go-session-id")
if not field then
    ngx.say('{"code":-1,"message":"请登录"}')
    return
end

ngx.req.set_header("X-API", "user/auth")
local res = ngx.location.capture("/user", {})
if not res then
    ngx.say('{"code":-1,"message":"请登录"}')
    return
end

if res then
    if tonumber(res.status) ~= tonumber(200) then
        ngx.say(res.body)
        return
    end
end

ngx.req.set_header("session", res.body)