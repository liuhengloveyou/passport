package user

import (
	"fmt"
	"github.com/liuhengloveyou/passport/v3/protos"
)

func normalizeUserExt(user *protos.User) {
	if user == nil {
		return
	}
	if user.Ext != nil {
		user.Ext = normalizeMap(user.Ext)
	}
	if user.Tenant != nil && user.Tenant.Configuration != nil && user.Tenant.Configuration.More != nil {
		user.Tenant.Configuration.More = normalizeMap(user.Tenant.Configuration.More)
	}
	if user.Departments != nil {
		for i := range user.Departments {
			if user.Departments[i].Config != nil {
				user.Departments[i].Config = normalizeMap(user.Departments[i].Config)
			}
		}
	}
}

func normalizeMap(input map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(input))
	for k, v := range input {
		out[k] = normalizeValue(v)
	}
	return out
}

func normalizeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return normalizeMap(v)
	case map[interface{}]interface{}:
		converted := make(map[string]interface{}, len(v))
		for key, val := range v {
			converted[fmt.Sprintf("%v", key)] = normalizeValue(val)
		}
		return converted
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			out[i] = normalizeValue(item)
		}
		return out
	default:
		return v
	}
}
