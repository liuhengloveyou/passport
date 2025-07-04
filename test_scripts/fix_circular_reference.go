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

	// 打印当前的循环引用关系
	fmt.Println("查询10002的所有祖先:")
	rows, err := tx.Query(context.Background(), 
		"SELECT ancestor_id, depth FROM tenant_closure WHERE descendant_id = 10002 ORDER BY depth")
	if err != nil {
		log.Fatalf("查询10002的祖先失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("祖先ID\t深度")
	for rows.Next() {
		var ancestorID uint64
		var depth int
		if err := rows.Scan(&ancestorID, &depth); err != nil {
			log.Fatalf("扫描结果失败: %v", err)
		}
		fmt.Printf("%d\t%d\n", ancestorID, depth)
	}

	fmt.Println("\n查询10105的所有祖先:")
	rows, err = tx.Query(context.Background(), 
		"SELECT ancestor_id, depth FROM tenant_closure WHERE descendant_id = 10105 ORDER BY depth")
	if err != nil {
		log.Fatalf("查询10105的祖先失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("祖先ID\t深度")
	for rows.Next() {
		var ancestorID uint64
		var depth int
		if err := rows.Scan(&ancestorID, &depth); err != nil {
			log.Fatalf("扫描结果失败: %v", err)
		}
		fmt.Printf("%d\t%d\n", ancestorID, depth)
	}

	// 查看循环引用的详细情况
	fmt.Println("\n查询循环引用的详细情况:")
	rows, err = tx.Query(context.Background(), `
		WITH RECURSIVE cycle_detection AS (
			SELECT descendant_id AS id, ancestor_id AS parent_id, ARRAY[descendant_id] AS path
			FROM tenant_closure
			WHERE depth = 1
			UNION ALL
			SELECT cd.id, tc.ancestor_id, cd.path || tc.ancestor_id
			FROM tenant_closure tc
			JOIN cycle_detection cd ON cd.parent_id = tc.descendant_id
			WHERE depth = 1 AND NOT tc.ancestor_id = ANY(cd.path)
		)
		SELECT id, parent_id, path
		FROM cycle_detection
		WHERE parent_id = ANY(path)
	`)
	if err != nil {
		log.Fatalf("查询循环引用详情失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("节点ID\t父节点ID\t路径")
	for rows.Next() {
		var nodeID, parentID uint64
		var path []uint64
		if err := rows.Scan(&nodeID, &parentID, &path); err != nil {
			log.Fatalf("扫描结果失败: %v", err)
		}
		fmt.Printf("%d\t%d\t%v\n", nodeID, parentID, path)
	}

	// 修复循环引用 - 更彻底的方法
	fmt.Println("\n修复循环引用 - 更彻底的方法...")
	
	// 1. 删除所有涉及10002和10105的关系
	result1, err := tx.Exec(context.Background(), 
		"DELETE FROM tenant_closure WHERE (ancestor_id = 10002 AND descendant_id = 10105) OR (ancestor_id = 10105 AND descendant_id = 10002)")
	if err != nil {
		log.Fatalf("删除10002和10105之间的关系失败: %v", err)
	}
	fmt.Printf("删除了 %d 条10002和10105之间的关系\n", result1.RowsAffected())

	// 2. 删除10105的所有祖先关系
	result2, err := tx.Exec(context.Background(), 
		"DELETE FROM tenant_closure WHERE descendant_id = 10105 AND ancestor_id != 10105")
	if err != nil {
		log.Fatalf("删除10105的所有祖先关系失败: %v", err)
	}
	fmt.Printf("删除了 %d 条10105的祖先关系\n", result2.RowsAffected())

	// 3. 重新建立10105的正确关系 - 假设10108是10105的父节点
	result3, err := tx.Exec(context.Background(), 
		"INSERT INTO tenant_closure (ancestor_id, descendant_id, depth) VALUES (10108, 10105, 1) ON CONFLICT (ancestor_id, descendant_id) DO UPDATE SET depth = 1")
	if err != nil {
		log.Fatalf("重新建立10105与10108的关系失败: %v", err)
	}
	fmt.Printf("重新建立了 %d 条10105与10108的关系\n", result3.RowsAffected())

	// 4. 修复传递关系
	result4, err := tx.Exec(context.Background(), `
		INSERT INTO tenant_closure (ancestor_id, descendant_id, depth)
		SELECT a.ancestor_id, b.descendant_id, a.depth + b.depth
		FROM tenant_closure a
		JOIN tenant_closure b ON a.descendant_id = b.ancestor_id
		WHERE a.ancestor_id != b.descendant_id
		ON CONFLICT (ancestor_id, descendant_id) DO NOTHING
	`)
	if err != nil {
		log.Fatalf("修复传递关系失败: %v", err)
	}
	fmt.Printf("修复了 %d 条传递关系\n", result4.RowsAffected())

	// 提交事务
	fmt.Println("提交事务...")
	err = tx.Commit(context.Background())
	if err != nil {
		log.Fatalf("提交事务失败: %v", err)
	}

	fmt.Println("循环引用修复成功!")
}