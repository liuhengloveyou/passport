-- 这里是配置
local conf = {
    passportUri = "http://127.0.0.1:10000/",
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
ngx.log(ngx.ERR, "cookie: ", token)
if not token then

    -- 为什么取不到token呢？打印所有HTTP头
    local h, err = ngx.req.get_headers()
    for k, v in pairs(h) do
         ngx.log(ngx.ERR, "header: ", k, ": ", v)
    end

    ngx.exit(ngx.HTTP_UNAUTHORIZED)
end

local xRequestedBy = ngx.var.xRequestedBy -- 可以自己设置接口名
if not xRequestedBy then
    xRequestedBy = ngx.var.uri
end
ngx.log(ngx.ERR, "api: ", xRequestedBy)

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
ngx.log(ngx.ERR, "passport resp: ", res.status, " body: ", res.body)
if res and tonumber(res.status) ~= tonumber(200) then
    ngx.exit(res.status)
end

if res.body == nil or res.body == ngx.null or string.len(res.body) < 2 then
    ngx.log(ngx.ERR, "usercenter body nil")
    ngx.exit(ngx.HTTP_UNAUTHORIZED)
end

-- 只有某个租户可访问
local onlyTenant = ngx.var.onlyTenant
if tonumber(onlyTenant) ~= nil and tonumber(onlyTenant) > 0 then
    local sessJson = require("cjson").decode(res.body)
    if tonumber(onlyTenant) ~= tonumber(sessJson["data"]["tenant_id"]) then
        ngx.exit(ngx.HTTP_UNAUTHORIZED)
    end
end

ngx.log(ngx.ERR, "passport auth OK: ", res.status, " body: ", res.body)
ngx.req.set_header("session", res.body)
ngx.req.clear_header("X-Requested-By")
