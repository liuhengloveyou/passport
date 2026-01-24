package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
)

func main() {
	// 设置日志
	err := common.InitLog("/tmp", "debug")
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 连接数据库
	connStr := "host=localhost user=pcdn password=pcdn12321 dbname=pcdn port=5432 sslmode=disable"
	var conn *pgx.Conn
	conn, err = pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	// 开始事务
	tx, err := conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("Unable to begin transaction: %v\n", err)
	}
	defer tx.Rollback(context.Background())

	// 执行移动操作
	tenantID := uint64(10106)
	newParentID := uint64(10002)

	// 打印移动前的状态
	printTenantClosureState(tx, tenantID)

	// 执行移动操作
	err = dao.TenantClosureUpdateSubtreeV2(tx, tenantID, newParentID)
	if err != nil {
		log.Fatalf("Failed to move tenant: %v\n", err)
	}

	// 注意：tenants表中已经没有parent_id字段，所以不需要更新tenants表
	// _, err = tx.Exec(context.Background(), "UPDATE tenants SET parent_id = $1 WHERE id = $2", newParentID, tenantID)
	// if err != nil {
	// 	log.Fatalf("Failed to update parent_id: %v\n", err)
	// }

	// 打印移动后的状态
	printTenantClosureState(tx, tenantID)

	// 提交事务
	err = tx.Commit(context.Background())
	if err != nil {
		log.Fatalf("Failed to commit transaction: %v\n", err)
	}

	fmt.Println("Tenant moved successfully!")
}

func printTenantClosureState(tx pgx.Tx, tenantID uint64) {
	rows, err := tx.Query(context.Background(), "SELECT ancestor_id, descendant_id, depth FROM tenant_closure WHERE descendant_id = $1 ORDER BY ancestor_id, depth", tenantID)
	if err != nil {
		log.Fatalf("Failed to query tenant_closure: %v\n", err)
	}
	defer rows.Close()

	fmt.Printf("Tenant %d closure state:\n", tenantID)
	fmt.Println("ancestor_id | descendant_id | depth")
	fmt.Println("-----------+---------------+-------")

	for rows.Next() {
		var ancestorID, descendantID uint64
		var depth int
		err := rows.Scan(&ancestorID, &descendantID, &depth)
		if err != nil {
			log.Fatalf("Failed to scan row: %v\n", err)
		}
		fmt.Printf("%10d | %13d | %5d\n", ancestorID, descendantID, depth)
	}
	fmt.Println()
}