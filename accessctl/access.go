package accessctl

import (
	"fmt"
)

func AddRoleForUser(uid uint64, role string) (err error) {
	return addRoleForUser(genUserByUID(uid), role)
}

func DeleteRoleForUser(uid uint64, role string) (err error) {
	return deleteRoleForUser(genUserByUID(uid), role)
}

func AddPolicy(uid uint64, obj, act string) (err error) {
	return addPolicy(genUserByUID(uid), obj, act)
}

func RemovePolicy(uid uint64, obj, act string) (err error) {
	return removePolicy(genUserByUID(uid), obj, act)
}

func genUserByUID(uid uint64) string {
	return fmt.Sprintf("UID-%v", uid)
}