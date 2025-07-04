package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/liuhengloveyou/passport/common"
	"github.com/liuhengloveyou/passport/dao"
)

func main() {
	// 连接数据库
	dbPool, err := pgxpool.New(context.Background(), "host=localhost user=pcdn password=pcdn12321 dbname=pcdn port=5432 sslmode=disable TimeZone=Asia/Shanghai")
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}
	defer dbPool.Close()

	// 设置日志
	// 使用common.InitLog初始化日志
	err = common.InitLog("/tmp", "debug")
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 测试移动租户
	tenantID := uint64(10106)   // 要移动的租户ID
	newParentID := uint64(10002) // 新的父租户ID

	// 开始事务
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		log.Fatalf("无法开始事务: %v", err)
	}
	defer tx.Rollback(context.Background())

	// 打印移动前的关系
	printTenantRelations(tx, tenantID)

	// 执行移动操作
	fmt.Printf("正在将租户 %d 移动到父租户 %d 下...\n", tenantID, newParentID)
	err = dao.TenantClosureUpdateSubtreeV2Safe(tx, tenantID, newParentID)
	if err != nil {
		log.Fatalf("移动租户失败: %v", err)
	}

	// 打印移动后的关系
	fmt.Println("移动后的关系:")
	printTenantRelations(tx, tenantID)

	// 提交事务
	fmt.Println("提交事务...")
	err = tx.Commit(context.Background())
	if err != nil {
		log.Fatalf("提交事务失败: %v", err)
	}

	fmt.Println("租户移动成功!")
}

// 打印租户的关系
func printTenantRelations(tx pgx.Tx, tenantID uint64) {
	// 打印租户的所有祖先
	rows, err := tx.Query(context.Background(), 
		"SELECT ancestor_id, depth FROM tenant_closure WHERE descendant_id = $1 ORDER BY depth", 
		tenantID)
	if err != nil {
		log.Fatalf("查询祖先关系失败: %v", err)
	}
	defer rows.Close()

	fmt.Printf("租户 %d 的祖先关系:\n", tenantID)
	fmt.Println("祖先ID\t深度")
	for rows.Next() {
		var ancestorID uint64
		var depth int
		if err := rows.Scan(&ancestorID, &depth); err != nil {
			log.Fatalf("扫描结果失败: %v", err)
		}
		fmt.Printf("%d\t%d\n", ancestorID, depth)
	}
	fmt.Println()

	// 打印租户的所有子孙
	rows, err = tx.Query(context.Background(), 
		"SELECT descendant_id, depth FROM tenant_closure WHERE ancestor_id = $1 AND descendant_id != $1 ORDER BY depth", 
		tenantID)
	if err != nil {
		log.Fatalf("查询子孙关系失败: %v", err)
	}
	defer rows.Close()

	fmt.Printf("租户 %d 的子孙关系:\n", tenantID)
	fmt.Println("子孙ID\t深度")
	for rows.Next() {
		var descendantID uint64
		var depth int
		if err := rows.Scan(&descendantID, &depth); err != nil {
			log.Fatalf("扫描结果失败: %v", err)
		}
		fmt.Printf("%d\t%d\n", descendantID, depth)
	}
	fmt.Println()
}