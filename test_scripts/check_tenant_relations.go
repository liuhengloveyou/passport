package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/liuhengloveyou/passport/common"
)

func main() {
	// 连接数据库
	dbPool, err := pgxpool.New(context.Background(), "host=localhost user=pcdn password=pcdn12321 dbname=pcdn port=5432 sslmode=disable TimeZone=Asia/Shanghai")
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}
	defer dbPool.Close()

	// 设置日志
	err = common.InitLog("/tmp", "debug")
	if err != nil {
		log.Fatalf("初始化日志失败: %v", err)
	}

	// 开始事务
	tx, err := dbPool.Begin(context.Background())
	if err != nil {
		log.Fatalf("无法开始事务: %v", err)
	}
	defer tx.Rollback(context.Background())

	// 查询所有depth=1的关系
	fmt.Println("所有直接父子关系(depth=1):")
	rows, err := tx.Query(context.Background(), 
		"SELECT ancestor_id, descendant_id FROM tenant_closure WHERE depth = 1 ORDER BY ancestor_id, descendant_id")
	if err != nil {
		log.Fatalf("查询直接父子关系失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("父节点ID\t子节点ID")
	for rows.Next() {
		var ancestorID, descendantID uint64
		if err := rows.Scan(&ancestorID, &descendantID); err != nil {
			log.Fatalf("扫描结果失败: %v", err)
		}
		fmt.Printf("%d\t%d\n", ancestorID, descendantID)
	}

	// 查询10002和10105的所有关系
	fmt.Println("\n10002和10105之间的所有关系:")
	rows, err = tx.Query(context.Background(), 
		"SELECT ancestor_id, descendant_id, depth FROM tenant_closure WHERE (ancestor_id IN (10002, 10105) OR descendant_id IN (10002, 10105)) ORDER BY ancestor_id, descendant_id, depth")
	if err != nil {
		log.Fatalf("查询10002和10105之间的关系失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("祖先ID\t后代ID\t深度")
	for rows.Next() {
		var ancestorID, descendantID uint64
		var depth int
		if err := rows.Scan(&ancestorID, &descendantID, &depth); err != nil {
			log.Fatalf("扫描结果失败: %v", err)
		}
		fmt.Printf("%d\t%d\t%d\n", ancestorID, descendantID, depth)
	}

	// 回滚事务（只是查询，不需要提交）
	tx.Rollback(context.Background())
}