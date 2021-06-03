local conf = {
    passportUri = "http://127.0.0.1:8001/",
    passportHost = "demo.passport.com"
}

local ngx = ngx
local ck = require "resty.cookie"
local cookie, err = ck:new()
if not cookie then
    ngx.log(ngx.ERR, err)
    return
end

local token, err = cookie:get("go-session-id")
if not token then
    token = ngx.req.get_headers()["Access-Token"]
end
if not token then
    ngx.exit(ngx.HTTP_UNAUTHORIZED)
end

local xRequestedBy = ngx.var.xRequestedBy -- 可以自己设置接口名
if not xRequestedBy then
    xRequestedBy = ngx.var.uri
end
local httpc = require("resty.http").new()
local res, err = httpc:request_uri(conf.passportUri, {
    path = "/usercenter",
    headers = {
        ["Host"] = conf.passportHost,
        ["Content-Type"] = "application/json;charset=UTF-8",
        ["X-API"] = "user/auth",
        ["X-Requested-By"] = xRequestedBy,
        ["Cookie"] = "go-session-id="..token..";"
    },
})
if not res then
    ngx.log(ngx.ERR, "usercenter ERR: ", err)
    ngx.exit(ngx.HTTP_UNAUTHORIZED)
end
ngx.log(ngx.ERR, "passport resp: ", res.body)
if res and tonumber(res.status) ~= tonumber(200) then
    ngx.exit(res.status)
end

-- 只有某个租户可访问
local onlyTenant = ngx.var.onlyTenant
if tonumber(onlyTenant) ~= nil and tonumber(onlyTenant) > 0 then
    local sessJson = require("cjson").decode(res.body)
    if tonumber(onlyTenant) ~= tonumber(sessJson["data"]["tenant_id"]) then
        ngx.exit(ngx.HTTP_UNAUTHORIZED)
    end
end

ngx.req.set_header("session", res.body)
ngx.req.clear_header("X-Requested-By")
