package service

import (
	"fmt"
	"strconv"

	"github.com/liuhengloveyou/passport/v4/accessctl"
	"github.com/liuhengloveyou/passport/v4/common"
	"github.com/liuhengloveyou/passport/v4/dao"
	"github.com/liuhengloveyou/passport/v4/protos"
)

// 数据范围：与 Casbin 功能权限并列。角色按模块配置 all / dept / self。
const (
	DataScopeLevelAll  = "all"
	DataScopeLevelDept = "dept"
	DataScopeLevelSelf = "self"

	// Department.config 中部门负责人 UID 列表（并入「本部门」可见部门）
	DepartmentLeaderUIDsKey = "leaderUids"
)

// DataScope 解析结果。Level=all 时 UIDs 为空表示不按人限制。
type DataScope struct {
	Level  string   `json:"level"`
	DepIDs []uint64 `json:"depIds,omitempty"`
	UIDs   []uint64 `json:"uids,omitempty"`
}

// ResolveDataScope 按角色为该 module 配置的数据范围解析。
// Casbin：data-scope/{module}/all|dept|self + GET；未配置则默认 self。
func ResolveDataScope(uid, tenantID, orgID uint64, module string) (DataScope, error) {
	if uid == 0 || tenantID == 0 || orgID == 0 {
		return DataScope{}, common.ErrParam
	}
	if module == "" {
		module = "default"
	}

	if isDataScopeRoot(uid, tenantID, orgID) {
		return DataScope{Level: DataScopeLevelAll}, nil
	}

	level := resolveConfiguredLevel(uid, tenantID, orgID, module)
	switch level {
	case DataScopeLevelAll:
		return DataScope{Level: DataScopeLevelAll}, nil
	case DataScopeLevelDept:
		depIDs, err := resolveUserDeptIDs(uid, tenantID, orgID)
		if err != nil {
			return DataScope{}, err
		}
		members, err := memberUIDsInDeps(tenantID, orgID, depIDs)
		if err != nil {
			return DataScope{}, err
		}
		if !containsUint64(members, uid) {
			members = append(members, uid)
		}
		return DataScope{Level: DataScopeLevelDept, DepIDs: depIDs, UIDs: members}, nil
	default:
		return DataScope{Level: DataScopeLevelSelf, UIDs: []uint64{uid}}, nil
	}
}

func resolveConfiguredLevel(uid, tenantID, orgID uint64, module string) string {
	for _, lv := range []string{DataScopeLevelAll, DataScopeLevelDept, DataScopeLevelSelf} {
		obj := fmt.Sprintf("data-scope/%s/%s", module, lv)
		ok, err := accessctl.Enforce(uid, tenantID, orgID, obj, "GET")
		if err != nil {
			common.Logger.Sugar().Warnf("ResolveDataScope Enforce: %v %v %v %v %v", uid, tenantID, orgID, obj, err)
			continue
		}
		if ok {
			return lv
		}
	}
	return DataScopeLevelSelf
}

func resolveUserDeptIDs(uid, tenantID, orgID uint64) ([]uint64, error) {
	user, err := dao.UserQueryByID(uid)
	if err != nil {
		common.Logger.Sugar().Errorf("resolveUserDeptIDs user: %v", err)
		return nil, common.ErrService
	}
	depIDs := make([]uint64, 0)
	seen := make(map[uint64]struct{})
	add := func(id uint64) {
		if id == 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		depIDs = append(depIDs, id)
	}

	deps, err := DepartmentFind(0, tenantID, orgID, 0, 0)
	if err != nil {
		common.Logger.Sugar().Errorf("resolveUserDeptIDs deps: %v", err)
		return nil, common.ErrService
	}
	orgDep := make(map[uint64]struct{}, len(deps))
	for i := range deps {
		orgDep[deps[i].Id] = struct{}{}
	}

	if user != nil && user.TenantID == tenantID {
		for _, id := range parseUint64Slice(user.Ext[protos.DepartmentExtKey]) {
			if _, ok := orgDep[id]; ok {
				add(id)
			}
		}
	}

	for i := range deps {
		if containsUint64(parseUint64Slice(deps[i].Config[DepartmentLeaderUIDsKey]), uid) {
			add(deps[i].Id)
		}
	}
	return depIDs, nil
}

func isDataScopeRoot(uid, tenantID, orgID uint64) bool {
	roles := accessctl.GetRoleForUserInDomain(uid, tenantID, orgID)
	for _, r := range roles {
		if r == "root" {
			return true
		}
	}
	return false
}

func memberUIDsInDeps(tenantID, orgID uint64, depIDs []uint64) ([]uint64, error) {
	if len(depIDs) == 0 {
		return nil, nil
	}
	depSet := make(map[uint64]struct{}, len(depIDs))
	for _, id := range depIDs {
		depSet[id] = struct{}{}
	}

	out := make([]uint64, 0)
	seen := make(map[uint64]struct{})
	const pageSize uint64 = 200
	for page := uint64(1); ; page++ {
		users, err := dao.UserQueryByOrg(tenantID, orgID, page, pageSize, "", nil)
		if err != nil {
			common.Logger.Sugar().Errorf("memberUIDsInDeps: %v", err)
			return nil, common.ErrService
		}
		if len(users) == 0 {
			break
		}
		for i := range users {
			uidsDeps := parseUint64Slice(users[i].Ext[protos.DepartmentExtKey])
			for _, d := range uidsDeps {
				if _, ok := depSet[d]; !ok {
					continue
				}
				if _, ok := seen[users[i].UID]; ok {
					break
				}
				seen[users[i].UID] = struct{}{}
				out = append(out, users[i].UID)
				break
			}
		}
		if uint64(len(users)) < pageSize {
			break
		}
	}
	return out, nil
}

func parseUint64Slice(v interface{}) []uint64 {
	if v == nil {
		return nil
	}
	switch arr := v.(type) {
	case []uint64:
		return append([]uint64(nil), arr...)
	case []interface{}:
		out := make([]uint64, 0, len(arr))
		for _, item := range arr {
			if n, ok := asUint64(item); ok && n > 0 {
				out = append(out, n)
			}
		}
		return out
	case []float64:
		out := make([]uint64, 0, len(arr))
		for _, item := range arr {
			if item > 0 {
				out = append(out, uint64(item))
			}
		}
		return out
	default:
		if n, ok := asUint64(v); ok && n > 0 {
			return []uint64{n}
		}
		return nil
	}
}

func asUint64(v interface{}) (uint64, bool) {
	switch x := v.(type) {
	case uint64:
		return x, true
	case int64:
		if x < 0 {
			return 0, false
		}
		return uint64(x), true
	case int:
		if x < 0 {
			return 0, false
		}
		return uint64(x), true
	case float64:
		if x < 0 {
			return 0, false
		}
		return uint64(x), true
	case string:
		n, err := strconv.ParseUint(x, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func containsUint64(list []uint64, target uint64) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
