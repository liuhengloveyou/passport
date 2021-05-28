local ck = require "resty.cookie"
local cookie, err = ck:new()
if not cookie then
    ngx.log(ngx.ERR, err)
    return
end

local field, err = cookie:get("go-session-id")
if not token then
    token = ngx.req.get_headers()["Access-Token"]
end
if not token then
    ngx.say('{"code":-1,"msg":"请登录"}')
    return
end

local xRequestedBy = ngx.var.xRequestBy
if not xRequestedBy then
    xRequestedBy = ngx.var.uri
end
local httpc = require("resty.http").new()
local res, err = httpc:request_uri("http://127.0.0.1:8001/", {
    path = "/usercenter",
    headers = {
        ["Host"] = "demo.passport.com",
        ["Content-Type"] = "application/json;charset-UTF-8",
        ["X-API"] = "user/auth",
        ["X-Requested-By"] = xRequestedBy,
        ["Cookie"] = "go-session-id="..token..";")
    },
})
if not res then
    ngx.log(ngx.ERR, "usercenter ERR: ", err)
    ngx.say('{"code":-1,"message":"请登录"}')
    return
end
ngx.log(ngx.ERR, "passport resp: ", res.body)
if res then
    if tonumber(res.status) ~= tonumber(200) then
        ngx.say(res.body)
        return
    end
end

ngx.req.set_header("session", res.body)
